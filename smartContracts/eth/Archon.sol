pragma solidity ^0.5.11;

import "./Ownable.sol";
import "./SafeMath.sol";
import "./Math.sol";

/* note: that the "//--TEST--//" guards below are specific to the 
 *   the comprehensive archon SC build suite. See archon documentation 
 *   for details
 */ 

contract Archon is Ownable {
  
  // state variables
  using SafeMath for uint;
  using SafeMath for uint64;
  using SafeMath for uint256;
  uint256 public registerCost; 
  uint256 public registerCostScalar;
  mapping (address => ArchonSPProfile) public spAddress2SPProfile; 
  
  mapping (bytes32 => UserProfile) public username2UserProfile;
  mapping (address => bytes32) public address2Username;
  mapping (bytes32 => address) public nodeID2Address;
  
  // scalars for proposed utility 
  uint256 public slaLevelScalar;
  uint256 public availableStorageScalar;
  uint256 public bandwidthScalar;
  uint256 public minAskPriceScalar;

  // global constants
  bytes32 zeroB32;
  uint256 zero256; 
  uint256 FFFFFFFFFFFFFFFF256;
  uint256 FFFFFFFF00000000256;
  uint256 bit256;
  uint256 byte256;
  uint oneUint;
  uint zeroUint;
  address zeroAddress;

  struct ArchonSPProfile {
    bytes32 params;
    bytes32 nodeID;
    uint256 stake;
    uint256 earnings;
    uint256 slash;
    uint256 remainingStorage;
    bool inGoodStanding;  
  }
  
  struct UserProfile {
    address usersAddress;
    bytes32 usersPublicKeyX;
    bytes32 usersPublicKeyY;
  }

  // events
  event LogRegisterSP(address sp);
  event LogUploadTx(address sp);
  event LogDepositReceived(address);
  event LogCostScalarUpdated();
  event LogDebug(uint256);
  
  constructor() public payable {
    // set global constants
    zeroB32 = bytes32(0);
    zero256 = uint256(0); 
    FFFFFFFFFFFFFFFF256 = uint256(0xFFFFFFFFFFFFFFFF);
    FFFFFFFF00000000256 = uint256(0xFFFFFFFF00000000);
    bit256 = uint256(1);
    byte256 = uint256(0xFF);
    oneUint = uint(1);
    zeroUint = uint(0);
    zeroAddress = address(0);
    // ownership initialization is handled by Ownable constructor 
    registerCost = uint256(1e11);  // goal 1e15
    registerCostScalar = uint256(10000); // goal 1e15
    // proposedUtility scalars
    slaLevelScalar = zero256;
    availableStorageScalar = zero256;
    bandwidthScalar = zero256;
    minAskPriceScalar = zero256; 
  }

  // fallback function
  function() external payable { 
    require(msg.data.length == uint(0));
    emit LogDepositReceived(msg.sender);
  }
  
  // external functions
  function registerSP(
    bytes32 params, 
    bytes32 nodeID,
    bytes32 hardwareProof
    //--TEST--// , uint256[] calldata testVals //  
  ) external payable {
    require(spAddress2SPProfile[msg.sender].params == zeroB32, 
            'this address is already a registeredSP');
    require(params != zeroB32, 'params must be nontrivial');
    require(nodeID != zeroB32, 'nodeID must be nontrivial');
    require(nodeID2Address[nodeID] == zeroAddress, 
            'nodeID cannot be associated with other sp');

    uint256 params256 = uint256(params); // reusable for efficiency
    uint256 totalCost = computeTotalCost(params256
                                         //--TEST--// , testVals
                                         );

    uint256 msgValue256 = uint256(msg.value);
    require(msgValue256 >= totalCost, 
            'insufficient registerSP payment');

    spAddress2SPProfile[msg.sender].params =  params;
    spAddress2SPProfile[msg.sender].nodeID = nodeID;
    spAddress2SPProfile[msg.sender].stake = msgValue256;
    spAddress2SPProfile[msg.sender].earnings = zero256;
    spAddress2SPProfile[msg.sender].slash = zero256;
    spAddress2SPProfile[msg.sender].remainingStorage = 
      (params256 & (FFFFFFFFFFFFFFFF256 << 184)) >> 184; // pledgedStorage  
    spAddress2SPProfile[msg.sender].inGoodStanding = true;
    // note: interestingly, EVM will not let contract owner register as sp
    // in that for contract owner the inGoodStanding will not be set to true
    
    nodeID2Address[nodeID] = msg.sender; 
    emit LogRegisterSP(msg.sender); 
  
    /* trivially "use" variables to keep compiler quiet
     * recall: we include these unused params so their values may be
     * accessible with the scanning logs wrt the address of Storage Provider. 
     *  Storage Hack!
     */
    bytes32 trivialBuffer = hardwareProof;
    trivialBuffer = trivialBuffer;
  }

  function proposeUpload(
    bytes32 hashedArchonFilepath, 
    bytes32 containerSignatureR,
    bytes32 containerSignatureS,
    bytes32 params, // has V
    uint64 shardsize,
    address[] calldata archonSPs
    //--TEST--// , uint256[] calldata testVals //  
  ) external payable {
    require(address2Username[msg.sender] != zeroB32, 
            'msg.sender address must map to nontrivial username');
    require(archonSPs.length > zeroUint, 'payment recipients not specified');
    
    uint256 maxMinAskPrice = getMaxMinAskPrice(archonSPs
                                              //--TEST--// , testVals
                                              );
    // checking payment granularity
    uint256 msgValue256 = uint256(msg.value); 
    uint256 share = msgValue256.div(uint256(archonSPs.length));
    require(share.mul(uint256(archonSPs.length)) >= msgValue256, 
            'payment not divisible');
    require(shardsize > uint64(0), 'shardsize must be non-trivial');
    
    uint256 timeInMonths = (uint256(params) & (FFFFFFFF00000000256 << 192)) 
                            >> (192 + 32);
    require(timeInMonths > zero256, 'upload must be for at least one month');
    //--TEST--// require(timeInMonths == testVals[1], 
    //--TEST--//   'TEST FAILED. TIMEINMONTHS');

    uint256 shardsizeInMegaBytes = shardsize.div(uint256(1000000));
    if (shardsizeInMegaBytes == zero256) {
      shardsizeInMegaBytes = bit256; 
    }
    uint256 pmtToEachSP = shardsizeInMegaBytes.mul(maxMinAskPrice);
    pmtToEachSP = pmtToEachSP.mul(timeInMonths);

    require(share >= pmtToEachSP, 
            'insufficient payment. payment must meet maxMinAskPrice of SP set');
            
    // pay each archonSPs 
    for (uint i = 0; i < archonSPs.length; i = i.add(oneUint)) {
      require(spAddress2SPProfile[archonSPs[i]].inGoodStanding, 
              'archonSPs must be registered and in good standing');
      require(spAddress2SPProfile[archonSPs[i]].remainingStorage >= shardsize, 
        'SP remainingStorage < shardsize');
      spAddress2SPProfile[archonSPs[i]].remainingStorage = 
        spAddress2SPProfile[archonSPs[i]].remainingStorage.sub(shardsize);
      spAddress2SPProfile[archonSPs[i]].earnings = 
        spAddress2SPProfile[archonSPs[i]].earnings.add(share); 
    
      emit LogUploadTx(archonSPs[i]);
    }
  
    /* trivially "use" variables to keep compiler quiet
     * recall: we include these unused params so their values may be
     * accessible with the rpc call "getTransactionByHash". Storage Hack!
     */
    bytes32 trivialBuffer = hashedArchonFilepath;
    trivialBuffer = containerSignatureR;
    trivialBuffer = containerSignatureS;
    trivialBuffer = params;  
  }

  // public functions
  function registerUsername(
    bytes32 username, 
    bytes32 publicKeyX, 
    bytes32 publicKeyY
  ) public {
    require(address2Username[msg.sender] == zeroB32 
            && username2UserProfile[username].usersAddress == zeroAddress, 
            'address already registered');
    require(username != zeroB32 
            && publicKeyX != zeroB32 
            && publicKeyY != zeroB32, 
              'register username must have non-trivial parameters');
    username2UserProfile[username].usersAddress = msg.sender;
    username2UserProfile[username].usersPublicKeyX = publicKeyX; 
    username2UserProfile[username].usersPublicKeyY = publicKeyY; 
    address2Username[msg.sender] = username;
  }

  function slashStake(
    bytes32 hashReference,
    uint256 amountToSlash,
    address spToSlash
  ) public onlyOwner {
    // slash 
    spAddress2SPProfile[spToSlash].slash 
      = spAddress2SPProfile[spToSlash].slash.add(amountToSlash);
    uint256 totalCreditOfSP = spAddress2SPProfile[spToSlash].stake
      .add(spAddress2SPProfile[spToSlash].earnings);
    // determine if in good standing
    uint256 creditAfterSlash = zero256;
    if (totalCreditOfSP > spAddress2SPProfile[spToSlash].slash) {
      creditAfterSlash = totalCreditOfSP.sub(spAddress2SPProfile[spToSlash].slash);
    }
    if (spAddress2SPProfile[spToSlash].stake > creditAfterSlash) {
      spAddress2SPProfile[spToSlash].inGoodStanding = false;
      // sp needs at least "stake" to participate in storage network
      // sp cannot accept proposeUploads
      // sp cannot withdrawal
      // when sp unregisters, gets difference of slash and stake
    }    
    /* trivially "use" variables to keep compiler quiet
     * recall: we include these unused params so their values may be
     * accessible with the rpc call "getTransactionByHash". Storage Hack!
     */
    bytes32 trivialBuffer = hashReference;
    trivialBuffer = trivialBuffer;
  }

  function updateCostScalar(
    string memory scalarName, 
    uint256 newScalarValue
  ) public onlyOwner {
    bytes32 hashedScalarName = keccak256(bytes(scalarName));
    if (hashedScalarName == keccak256(bytes("registerCostScalar"))) { 
      registerCostScalar = newScalarValue;
    } else if (hashedScalarName == keccak256(bytes("slaLevelScalar"))) {
      slaLevelScalar = newScalarValue;
    } else if (hashedScalarName == keccak256(bytes("availableStorageScalar"))) {
      availableStorageScalar = newScalarValue;
    } else if (hashedScalarName == keccak256(bytes("bandwidthScalar"))) {
      bandwidthScalar = newScalarValue;
    } else if (hashedScalarName == keccak256(bytes("minAskPriceScalar"))) {
      minAskPriceScalar = newScalarValue;
    } else {
      revert("invalid scalarName");
    }
    emit LogCostScalarUpdated(); 
  }

  
  function unregisterSP() public {
    nodeID2Address[spAddress2SPProfile[msg.sender].nodeID] = zeroAddress;
    
    spAddress2SPProfile[msg.sender].params = zeroB32;
    spAddress2SPProfile[msg.sender].nodeID = zeroB32;
    
    uint256 earnings = spAddress2SPProfile[msg.sender].earnings; 
    spAddress2SPProfile[msg.sender].earnings = zero256;
    
    uint256 settlement = zero256;
    if (spAddress2SPProfile[msg.sender].inGoodStanding) {
      uint256 totalCreditOfSP 
        = earnings.add(spAddress2SPProfile[msg.sender].stake);
      
      settlement = totalCreditOfSP.sub(spAddress2SPProfile[msg.sender].slash); 
    } else {
      if (spAddress2SPProfile[msg.sender].stake 
          > spAddress2SPProfile[msg.sender].slash) {
        settlement =  spAddress2SPProfile[msg.sender].stake
          .sub(spAddress2SPProfile[msg.sender].slash);
      }
    }
    
    spAddress2SPProfile[msg.sender].stake = zero256;
    spAddress2SPProfile[msg.sender].slash = zero256;
    spAddress2SPProfile[msg.sender].inGoodStanding = false;
    
    (bool success, ) = msg.sender.call.value(settlement)("");
    require(success, "Transfer failed.");
  } 
  
  function archonSPWithdrawal() public { 
    // follows standard to prevent re-entrancy
    require(spAddress2SPProfile[msg.sender].inGoodStanding, 
            'no funds to withdraw, not in good standing');
    uint256 totalCreditOfSP = spAddress2SPProfile[msg.sender].stake
      .add(spAddress2SPProfile[msg.sender].earnings);

    uint256 creditAfterSlash = totalCreditOfSP
      .sub(spAddress2SPProfile[msg.sender].slash);
    
    uint256 amountToWithdrawal = zero256;
    if (creditAfterSlash > spAddress2SPProfile[msg.sender].stake) {
      amountToWithdrawal = creditAfterSlash
        .sub(spAddress2SPProfile[msg.sender].stake);
      spAddress2SPProfile[msg.sender].earnings = zero256; 
    }

    (bool success, ) = msg.sender.call.value(amountToWithdrawal)("");
    require(success, "Transfer failed.");
  }
  
  // internal functions

  // private functions
  function computeTotalCost(
    uint256 params256 
    //--TEST--// , uint256[] memory testVals //  
  ) view private returns (uint256) {
    uint256 totalCost = zero256;
    totalCost = totalCost.add(registerCost.mul(registerCostScalar));
    
    
    // slaLevel 
    uint256 rightHandSummand = params256 >> 248; // this gives params[0]
    //--TEST--// require(rightHandSummand == testVals[0], 
    //--TEST--//        'TEST FAILED. SLALEVEL FAILED');
    totalCost = totalCost.add(rightHandSummand.mul(slaLevelScalar));
    
    // avaStorage
    rightHandSummand = (params256 & (FFFFFFFFFFFFFFFF256 << 184)) >> 184; 
    //--TEST--// require(rightHandSummand == testVals[1], 
    //--TEST--//  'TEST FAILED. AVAILABLESTORAGE FAILED');
    totalCost = totalCost.add(rightHandSummand.mul(availableStorageScalar));
   
    rightHandSummand = (params256 & (FFFFFFFFFFFFFFFF256 << 120)) >> 120; 
    //--TEST--// require(rightHandSummand == testVals[2], 
    //--TEST--//  'TEST FAILED. BANDWIDTH FAILED');
    totalCost = totalCost.add(rightHandSummand.mul(bandwidthScalar));
    
    // minAskPrice
    rightHandSummand = (params256 & (FFFFFFFFFFFFFFFF256 << 56)) >> 56;
    //--TEST--// require(rightHandSummand == testVals[3], 
    //--TEST--//  'TEST FAILED. MIN ASK PRICE');
    totalCost = totalCost.add(rightHandSummand.mul(minAskPriceScalar));

    return totalCost;
  }

  function getMaxMinAskPrice(
    address[] memory archonSPs
    //--TEST--// , uint256[] memory testVals //
  ) view private returns (uint256) {
    uint256 maxMinAskPrice = zero256;
    for (uint i = 0; i < archonSPs.length; i = i.add(oneUint)) {
      uint256 minAskPrice = (uint256(spAddress2SPProfile[archonSPs[i]].params) 
                              & (FFFFFFFFFFFFFFFF256 << 56)) >> 56;
      if (minAskPrice > maxMinAskPrice) {
        maxMinAskPrice = minAskPrice;
      }   
    }
    //--TEST--// require(maxMinAskPrice == testVals[0], 'TEST FAILED, maxMinAskPrice incorrect');
    return maxMinAskPrice;
  }
}
