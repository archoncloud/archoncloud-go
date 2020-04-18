// ? tests

var Int64 = require('node-int64');

module.exports = () => {
var zeroPad = function(input, padLength) {
  var zero = '0';
  var ret = input;
  while (ret.length < padLength) {
    ret = zero + ret;
  }
  return ret;
}

var TestProposeUploadParseParams = function () {}
TestProposeUploadParseParams.prototype.run = function(testParams) {
  const abi = testParams.abi;
  const code = testParams.code;
  var wallet = testParams.wallets[0];
  var wallets = testParams.wallets;
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
              console.log(receipt.contractAddress);
              initialize(receipt.contractAddress);
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

  var initialize = (contractAddress) => {
    var myContract = new web3.eth.Contract(abi, contractAddress, {defaultAccount: wallet.address});
    // register sps
    var hardwareProof = web3.utils.sha3("some preimage of hardwareProof");
    hardwareProof = hardwareProof.replace("0x", "");
    hardwareProof = Buffer.from(hardwareProof, 'hex');

    var testVals = [];
    var regVals = []; // to keep tests quiet on registerSP so that we can test proposeUpload
    // 0 slaLevel
    const maxSLALevel = 8;
    var slaLevel = Math.floor(Math.random() * maxSLALevel);
    regVals.push(slaLevel);
    const encSlaLevel = zeroPad(slaLevel.toString('16'), 2); 
    // 1 availableStorage
    var availableStorage = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    regVals.push(availableStorage);
    var encAvailableStorage = zeroPad(availableStorage.toString('16'), 16);
    // 2 bandwidth
    var bandwidth = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    regVals.push(bandwidth);
    var encBandwidth = zeroPad(bandwidth.toString('16'), 16);
   

    // 4 country code
    var cc0 = 233;
    var cc1 = 1;
    var cc = zeroPad(cc0.toString('16'), 2) + zeroPad(cc1.toString('16'), 2);
    
    var archonSPs = []; 
    var numSPsToRegister = 5;// FIXME TEST VAL 3;
    for (var i = 1; i < numSPsToRegister + 1; i++) {
       var archonSP = wallets[i].address; 
       archonSPs.push(archonSP);//archonSP, 'hex');
    } 

    // register3 nontrivial archonSPs start
    var numSPsRegistered = 0;
    var maxMinAskPrice = 0;
    for (var i = 1; i < numSPsToRegister + 1; i++) { 
      // 5 min ask price
      var minAskPrice = 4000000;// FIXME this is test val//parseInt(BigInt(Math.floor(4000000/*Number.MAX_SAFE_INTEGER*/ / 6)).toString(), 16);
      /*console.log("big int");
      var big = BigInt("0xFFFFFFFFFFFFFFFFFFFFF");
      console.log({big});
      console.log(big.toString());*/
      var _regVals = regVals.slice();
      _regVals.push(minAskPrice);
      if (minAskPrice > maxMinAskPrice) {
        maxMinAskPrice = minAskPrice;
      }
      var encMinAskPrice = zeroPad(minAskPrice.toString('16'), 16);
      var extraByte = "00";
      var params = encSlaLevel + encAvailableStorage + encBandwidth + encMinAskPrice + cc + "0000000000";
      params = params.replace("0x", "");
      params = Buffer.from(params, 'hex');
      registerSP(i, params, _regVals);
    }
    var goodPmt = 1000000000000000; //
    function registerSP(i, params, regVals) { 
     var rand = Math.floor(Math.random() * 1000000000);
     var nodeID = web3.utils.sha3("some/nodeID" + wallets[i].address + rand.toString()); // nodeID must be unique to address and unregistered
     nodeID = nodeID.replace("0x", "");
     nodeID = Buffer.from(nodeID, 'hex');
     web3.eth.getTransactionCount(wallets[i].address, 'pending')
     .then(nonce => {
      var encoded = myContract.methods.registerSP(params, nodeID, hardwareProof, regVals).encodeABI();
      wallets[i].signTransaction({from: wallets[i].address, to: contractAddress, gas: 6721974, nonce: nonce, data: encoded, value: goodPmt})// here
        .then(ret => {
        //console.log({ret});
        web3.eth.sendSignedTransaction(ret.rawTransaction)
        .on('transactionHash', (transactionHash) => {
          //console.log({transactionHash});
          numSPsRegistered++;
          //console.log("registered nontrivial archonSP");
        })
        .on('receipt', (receipt) => {
          //console.log(receipt)
        })
        .catch(err => { 
          console.error(err);
          if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
           setTimeout(() => {registerSP(i)}, 100);
          }
        });
        });
     });
    }
    // register3 nontrivial archonSPs end
    
    var numNamesRegistered = 0;
    var numNamesToRegister = 3;
    var registerUsername = function(i) {
      var username = web3.utils.sha3("generating random username" + i);
      username = username.replace("0x", "");
      username = Buffer.from(username, 'hex');
      var publicKeyX = web3.utils.sha3("generating fake publicKeyX" + i);
      publicKeyX = publicKeyX.replace("0x", "");
      publicKeyX = Buffer.from(publicKeyX, 'hex');
      var publicKeyY = web3.utils.sha3("generating fake publicKeyY" + i); // TODO MAKE REAL LATERS
      publicKeyY = publicKeyY.replace("0x", "");
      publicKeyY = Buffer.from(publicKeyY, 'hex');
      web3.eth.getTransactionCount(wallets[i].address, 'pending')
      .then(nonce => {
        /*myContract.methods.registerSP(params).estimateGas()
        .then(est => {
          console.log(est);*/
          var encoded = myContract.methods.registerUsername(username, publicKeyX, publicKeyY).encodeABI();
          wallets[i].signTransaction({from: wallets[i].address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded})
            .then(ret => {
            //console.log({ret});
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('transactionHash', (transactionHash) => {
            })
            .on('receipt', (receipt) => {
              numNamesRegistered++; 
            })
            .catch(err => { 
              if (err.toString().indexOf("correct nonce") > -1) {
                setTimeout(() => {registerUsername(i)}, 100);
              }
            });
            });
        //});
      })
      .catch(err => {
        if (err.toString().indexOf("correct nonce") > -1) {
          setTimeout(() => {registerUsername(i)}, 100);
        }
      });

    }
   for (var i = 0; i < numNamesToRegister; i++) {
     registerUsername(i); 
   }


    var waitForSPsToRegisterInterval = setInterval(() => {
      if (numSPsRegistered === numSPsToRegister
          && numNamesRegistered === numNamesToRegister) {
        testVals.push(maxMinAskPrice.toString());
        runTests(contractAddress, archonSPs, testVals);
        clearInterval(waitForSPsToRegisterInterval); // clearing self
      } 
    }, 200);

  }
  
  var runTests = (contractAddress, archonSPs, testVals) => {
    var myContract = new web3.eth.Contract(abi, contractAddress, {defaultAccount: wallet.address});
    // TESTS
    
    var randByte = function() {
      return Math.floor(Math.random() * 256);
    }
    // filesize
    var genParamsAndShardsize = function() {
      var petaByte = BigInt(1000000000000/*000*/); // short for now
      var filesize = petaByte;//Math.floor(Math.random() * petaByte);
      //console.log(filesize);
      var encFilesize = zeroPad(filesize.toString('16'), 16);
      //console.log(encFilesize);
      // encryptionType
      var encryptionType = randByte();
      var encEncryptionType = zeroPad(encryptionType.toString('16'), 2);
      // compressionType
      var compressionType = randByte();
      var encCompressionType = zeroPad(compressionType.toString('16'), 2);
      // shardContainerType
      var shardContainerType = randByte();
      var encShardContainerType = zeroPad(shardContainerType.toString('16'), 2);
      // erasureCodeType
      var erasureCodeType = randByte();
      var encErasureCodeType = zeroPad(erasureCodeType.toString('16'), 2);

      var timeInMonths = 1;// FIXME TEST VALUE Math.floor(Math.random() * 11) + 1; // avoid 0
      if (testVals.length == 2) {
        testVals.pop(); 
        // hack since genParamsAndShardsize() would put extra vals in testVals
      }
      testVals.push(timeInMonths);
      var encTimeInMonths = zeroPad(timeInMonths.toString('16'), 8);
      var frontPad = "";
      for (var i = 0; i < 12; i++) {
        frontPad += zeroPad(randByte().toString('16'), 2);
      }
      var backPad = "";
      for (var i = 0; i < 3; i++) {
        backPad += zeroPad(randByte().toString('16'), 2);
      }

      var params = encTimeInMonths
                     + frontPad 
                     + encFilesize 
                     + encEncryptionType 
                     + encCompressionType 
                     + encShardContainerType
                     + encErasureCodeType
                     + backPad;
      params = Buffer.from(params, 'hex');
      var shardsize = BigInt(6150);// FIXME TEST VALUE//filesize / BigInt(6);
      return {params, shardsize, timeInMonths};
    }
    var hashedArchonFilepath = web3.utils.sha3("/some/random/archon/cloud/filepath");
    hashedArchonFilepath = hashedArchonFilepath.replace("0x", "");
    hashedArchonFilepath = Buffer.from(hashedArchonFilepath, 'hex');
    var containerSignatureR = web3.utils.sha3("some preimage of a signature of some random container");
    containerSignatureR = containerSignatureR.replace("0x", "");
    containerSignatureR = Buffer.from(containerSignatureR, 'hex');
    var containerSignatureS = containerSignatureR;
    var gas = 6721974; // specific to ganache

    //var correctedPmt = "100000000000002"; // exceeds possible random test values

    var testProposeUploadParseParams = function() {
      var {params, shardsize, timeInMonths} = genParamsAndShardsize();
      var maxMinSPBid = testVals[0];
      var shardsizeInMegabytes = shardsize / BigInt(1000000);
      if (shardsizeInMegabytes === BigInt(0)) {
        shardsizeInMegabytes = BigInt(1);
      }
      var pmt = BigInt(maxMinSPBid) * BigInt(archonSPs.length) * shardsizeInMegabytes * BigInt(timeInMonths);
      var correctedPmt = pmt;
      var r = parseInt(correctedPmt.toString(), 16) % archonSPs.length;
      if (r != 0) {
        correctedPmt += BigInt(archonSPs.length) - BigInt(r);
      }
      correctedPmt = 20000000;// FIXME parseInt(correctedPmt.toString(), 16);
      var _shardsize = new Int64(parseInt(shardsize.toString(), 16));
      _shardsize = _shardsize + 0; // castes to uint64 hack
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
        console.log(correctedPmt, maxMinSPBid, archonSPs.length, shardsizeInMegabytes, timeInMonths);
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, _shardsize, archonSPs, testVals).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: correctedPmt})// here
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
         })
         .on('receipt', (receipt) => {
           //console.log(receipt);
           testParams.testsPassed++; 
           console.log("passed testProposeUploadParams");
         })
         .catch(err => { 
           //console.log(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 || err.toString().indexOf("insufficient payment") > -1 ) {
            setTimeout(() => {testProposeUploadParseParams()}, 100);
         
          } else if (err.toString().indexOf("TEST FAILED") > -1) {
            console.log(err);
            testParams.testsFailed++;
            console.log("failed testRegisterSPParseParams");
          }
         });
         });
      })
      .catch(err => {
        //console.log(err);
        if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
          setTimeout(() => {testProposeUploadParseParams()}, 100);
        }
      });

    }
    testProposeUploadParseParams();
  };

};
  return new TestProposeUploadParseParams;
};// module.exports
