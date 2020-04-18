// ? tests
module.exports = () => {
var zeroPad = function(input, padLength) {
  var zero = '0';
  var ret = input;
  while (ret.length < padLength) {
    ret = zero + ret;
  }
  return ret;
}

var TestRegisterSP = function () {}
TestRegisterSP.prototype.run = function(testParams) {
  const abi = testParams.abi;
  const code = testParams.code;
  var wallet = testParams.wallets[0];
  var web3 = testParams.web3;
  
  var deployContract = () => {
    web3.eth.getTransactionCount(wallet.address, 'pending')
    .then(nonce => {
      web3.eth.estimateGas(
        {from: wallet.address, nonce: nonce, data: code}
      )
      .then( est => {
        fee = est + 10000;
        wallet.signTransaction({from:wallet.address, gas: fee, nonce: nonce, data: code})
        .then(ret => {
          web3.eth.sendSignedTransaction(ret.rawTransaction)
          .on('transactionHash', (transactionHash) => {
          })
          .on('receipt', (receipt) => {
              //subscribeToLogs(receipt.contractAddress);
              runTests(receipt.contractAddress);
          })
          .catch(err => {
            if (err.toString().indexOf("correct nonce") > -1) {
              deployContract();
            }
          });
          });
      })
      .catch(err => {
        if (err.toString().indexOf("correct nonce") > -1) {
          deployContract();
        }
      });
    });
  } 
  deployContract();
  
  var runTests = (contractAddress) => {
    var myContract = new web3.eth.Contract(abi, contractAddress, {defaultAccount: wallet.address});
    // TESTS
    
    var testVals = []; 
    // 0 slaLevel
    const maxSLALevel = 8;
    var slaLevel = Math.floor(Math.random() * maxSLALevel);
    testVals.push(slaLevel);
    const encSlaLevel = zeroPad(slaLevel.toString('16'), 2); 
    
    // 1 availableStorage
    var availableStorage = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    var encAvailableStorage = zeroPad(availableStorage.toString('16'), 16);
    testVals.push(availableStorage);
    // 2 bandwidth
    var bandwidth = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    var encBandwidth = zeroPad(bandwidth.toString('16'), 16);
    testVals.push(bandwidth);
    // 3 min ask price
    var minAskPrice = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    var encMinAskPrice = zeroPad(minAskPrice.toString('16'), 16);
    testVals.push(minAskPrice);
    // 4 country code
    var cc0 = 233;
    var cc1 = 1;
    var cc = zeroPad(cc0.toString('16'), 2) + zeroPad(cc1.toString('16'), 2);

    var params = encSlaLevel + encAvailableStorage + encBandwidth + encMinAskPrice + cc + "0000000000";
    params = Buffer.from(params, 'hex');
    var hardwareProof = web3.utils.sha3("some preimage of hardwareProof");
    hardwareProof = hardwareProof.replace("0x", "");
    hardwareProof = Buffer.from(hardwareProof, 'hex');

    var goodPmt = 1000000000000000;
    var badPmt = 999999999999999;

    var testRegisterSPParseParams = function() {
      var rand = Math.floor(Math.random() * 1000000000);
      var nodeID = web3.utils.sha3("some/nodeID" + wallet.address + rand.toString()); // nodeID must be unique to address and unregistered
      nodeID = nodeID.replace("0x", "");
      nodeID = Buffer.from(nodeID, 'hex');
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
        /*myContract.methods.registerSP(params).estimateGas()
        .then(est => {
          console.log(est);*/
          var encoded = myContract.methods.registerSP(params, nodeID, hardwareProof, testVals).encodeABI();
          wallet.signTransaction({from:wallet.address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded, value: goodPmt})// here
            .then(ret => {
            //console.log({ret});
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('transactionHash', (transactionHash) => {
              //console.log({transactionHash});
              testParams.testsPassed++; 
              console.log("passed testRegisterSPParseParams");
              //runTestSuite(); 
            })
            .on('receipt', (receipt) => {
              //console.log(receipt);
            })
            .catch(err => {
              if (err.toString().indexOf("correct nonce") > -1) {
                setTimeout(() => {testRegisterSPParseParams()}, 100);
              } else if (err.toString().indexOf("TEST FAILED") > -1) {
                testParams.testsFailed++;
                console.log("failed testRegisterSPParseParams");
              }
            });
            });
        //});
      })
      .catch(err => { 
        if (err.toString().indexOf("correct nonce") > -1) {
          setTimeout(() => {testRegisterSPParseParams()}, 100);
        }
      });
    }
    testRegisterSPParseParams();
  };

};
  return new TestRegisterSP;
};// module.exports
