// 8 but 1 is disabled
module.exports = () => {
var zeroPad = function(input, padLength) {
  var zero = '0';
  var ret = input;
  while (ret.length < padLength) {
    ret = zero + ret;
  }
  return ret;
}
var randByte = function() {
  return Math.floor(Math.random() * 256);
}

var TestProposeUpload = function() {}
TestProposeUpload.prototype.run = function(testParams) {
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
              console.log(receipt.contractAddress);
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
    // TEST PROPOSEUPLOAD 
    /*
     * proposeUpload(
     *  bytes32 hashedArchonFilepath,
     *  bytes32 containerSignature,
     *  bytes32 params,
     *  address[] calldata archonSPs
     * )
     */

    ///for sps 
    // 0 slaLevel
    const maxSLALevel = 8;
    var slaLevel = Math.floor(Math.random() * maxSLALevel);
    const encSlaLevel = zeroPad(slaLevel.toString('16'), 2); 
    // 1 availableStorage
    var availableStorage = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    var encAvailableStorage = zeroPad(availableStorage.toString('16'), 16);
    // 2 bandwidth
    var bandwidth = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    var encBandwidth = zeroPad(bandwidth.toString('16'), 16);
    // 3 min ask price
    var minAskPrice = Math.floor(Math.random() * (2 ** 8)/*Number.MAX_SAFE_INTEGER*/);
    var encMinAskPrice = zeroPad(minAskPrice.toString('16'), 16);
    // 4 country code
    var cc0 = 233;
    var cc1 = 1;
    var cc = zeroPad(cc0.toString('16'), 2) + zeroPad(cc1.toString('16'), 2);

    var spParams = encSlaLevel + encAvailableStorage + encBandwidth + encMinAskPrice + cc + "0000000000";
     ////////////////////// 
    spParams = Buffer.from(spParams, 'hex');

     var petaByte = 1000000000000000;
     var filesize = Math.floor(Math.random() * petaByte);
     var hashedArchonFilepath = web3.utils.sha3("/some/random/archon/cloud/filepath");
     hashedArchonFilepath = hashedArchonFilepath.replace("0x", "");
     hashedArchonFilepath = Buffer.from(hashedArchonFilepath, 'hex');
     var containerSignatureR = web3.utils.sha3("some preimage of a signature of some random container");
     containerSignatureR = containerSignatureR.replace("0x", "");
     containerSignatureR = Buffer.from(containerSignatureR, 'hex');
     var containerSignatureS = containerSignatureR;
     var timeInMonths = Math.floor(Math.random() * 12);
     var encTimeInMonths = zeroPad(timeInMonths.toString('16'), 8);
     var frontPad = "";
     for (var i = 0; i < 12; i++) {
       frontPad += zeroPad(randByte().toString('16'), 2);
     }
     var petaByte = 1000000000000000;
     var filesize = Math.floor(Math.random() * petaByte);
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
     var shardsize = Math.floor(filesize / 6);
     var hardwareProof = web3.utils.sha3("some preimage of hardwareProof");
     hardwareProof = hardwareProof.replace("0x", "");
     hardwareProof = Buffer.from(hardwareProof, 'hex');
     /*var username = web3.utils.sha3("generating random username");
     username = username.replace("0x", "");
     username = Buffer.from(username, 'hex');*/
     var gas = 6721974;

     var archonSPs = []; 
     var numSPsToRegister = 3;
     var numNamesToRegister = 3;
     for (var i = 1; i < numSPsToRegister + 1; i++) {
        var archonSP = wallets[i].address; 
        //archonSP = archonSP.replace("0x", "");
        archonSPs.push(archonSP);//archonSP, 'hex');
     } 


     // register3 nontrivial archonSPs start
     var goodPmt = 1000000000000000;
     var badPmt = 999999999999999;
     var numSPsRegistered = 0;
     for (var i = 1; i < numSPsToRegister + 1; i++) { 
      registerSP(i);
     }
     function registerSP(i) { 
       var rand = Math.floor(Math.random() * 1000000000);
       var nodeID = web3.utils.sha3("some/nodeID" + wallets[i].address + rand.toString()); // nodeID must be unique to address and unregistered
       nodeID = nodeID.replace("0x", "");
       nodeID = Buffer.from(nodeID, 'hex');
      web3.eth.getTransactionCount(wallets[i].address, 'pending')
      .then(nonce => {
       var encoded = myContract.methods.registerSP(spParams, nodeID, hardwareProof).encodeABI();
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
           //console.error(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
            setTimeout(() => {registerSP(i)}, 100);
           }
         });
         });
      });
     }
     // register3 nontrivial archonSPs end

      var numNamesRegistered = 0;
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
        testProposeUploadTx();
        clearInterval(waitForSPsToRegisterInterval); // clearing self
      } 
     }, 200);

     
     var testPaymentGranularityFinished = false;
     
    var testProposeUploadTx = function() {
        //testTrivialPayment(); disabled while spec stays at "free upload"
       testTrivialPaymentRecipients();
       testPaymentGranularity();
       testPaymentGranularityFalsePositive();  
       testRequireArchonSPIsRegistered();
       testRequireArchonSPIsRegisteredFalsePositive();
       testRequireShardsizeNontrivial();
     
       var waitForTestPaymentGranularityInterval = setInterval(() => {
          if (testPaymentGranularityFinished) {
            setTimeout(() => {testPaymentAccounted();}, 1000); // will use testPaymentGranularity pmt 
            clearInterval(waitForTestPaymentGranularityInterval); // clear self
          }
       }, 500); 
     }

     var testTrivialPayment = function() {
      // test require(msg.value > 0, 'payment must be non-trivial');
      var trivialPmt = 0;
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
       //console.log({nonce});
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, shardsize, archonSPs/*, username*/).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: trivialPmt})// here
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
           testParams.testsFailed++; 
           console.log("failed proposeUpload trivial pmt test");
         })
         .on('receipt', (receipt) => {
           //console.log(receipt)
         })
         .catch(err => { 
           //console.error(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
            setTimeout(() => {testTrivialPayment()}, 100);
           } else if (err.toString().indexOf('revert payment must be non-trivial') > -1) {
             testParams.testsPassed++; 
             console.log("passed proposeUpload trivial pmt test");
           }
         });
         });
      });
     }

     var testTrivialPaymentRecipients = function() {
       // test require(archonSPs.length > 0, 'payment recipients not specified');
      var trivialArchonSPs = [];
      var nonTrivialPmt = 1000000000000000;
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, shardsize, trivialArchonSPs/*, username*/).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: nonTrivialPmt})// here
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
           testParams.testsFailed++; 
           console.log("failed proposeUpload trivial pmt recipients");
         })
         .on('receipt', (receipt) => {
           console.log(receipt)
         })
         .catch(err => { 
           //console.log(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
            setTimeout(() => {testTrivialPaymentRecipients()}, 100);
           } else if (err.toString().indexOf('payment recipients not specified') > -1) {
             testParams.testsPassed++; 
             console.log("passed proposeUpload trivial pmt recipients");
           }
         });
         });
      })
      .catch(err => {
        if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
          setTimeout(() => {testTrivialPaymentRecipients()}, 100);
        }
      });
     }

     var testPaymentGranularity = function() {
      // test require(share.mul(archonSPs.length) >= msg.value, 'payment not divisible');
      var nonTrivialPmt = 1000000000000000;
      var correctedPmt = nonTrivialPmt;
      var r = correctedPmt % archonSPs.length;
      if (r != 0) {
        correctedPmt += archonSPs.length - r;
      }
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
       //console.log({nonce});
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, shardsize, archonSPs/*, username*/).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: correctedPmt})
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
           testParams.testsPassed++; 
           console.log("passed proposeUpload payment granularity");
           testPaymentGranularityFinished = true;
         })
         .on('receipt', (receipt) => {
           //console.log(receipt)
         })
         .catch(err => { 
           //console.log(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 /*|| err.toString().indexOf('revert archonSPs must be registered') > -1*/) {
            setTimeout(() => {testPaymentGranularity()}, 100);
           } else if (err.toString().indexOf('payment not divisible') > -1) {
             //console.log(err);
             testParams.testsFailed++; 
             console.log("failed proposeUpload payment granularity");
           }
         });
         });
      })
      .catch(err => {
        if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 /*|| err.toString().indexOf('revert archonSPs must be registered') > -1*/) {
          setTimeout(() => {testPaymentGranularity()}, 100);
        }
      });
     }
     
     var testPaymentGranularityFalsePositive = function() {
      // test require(share.mul(archonSPs.length) >= msg.value, 'payment not divisible');
      var nonTrivialPmt = 1000000000000000;
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
       //console.log({nonce});
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, shardsize, archonSPs/*, username*/).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: nonTrivialPmt})
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
           testParams.testsFailed++; 
           console.log("failed proposeUpload payment granularity (false positive)");
         })
         .on('receipt', (receipt) => {
           //console.log(receipt)
         })
         .catch(err => { 
           //console.error(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 || err.toString().indexOf('revert archonSPs must be registered and in good standing') > -1) {
            setTimeout(() => {testPaymentGranularityFalsePositive()}, 100);
           } else if (err.toString().indexOf('payment not divisible') > -1) {
             testParams.testsPassed++; 
             console.log("passed proposeUpload payment granularity (false positive)");
           }
         });
         });
      });
     }

     var testRequireArchonSPIsRegistered = function() {
        // test require(spAddress2Profile[archonSPs[i]].params != 0, 'archonSPs must be registered');
      var nonTrivialPmt = 1000000000000000;
      var correctedPmt = nonTrivialPmt;
      var r = correctedPmt % archonSPs.length;
      if (r != 0) {
        correctedPmt += archonSPs.length - r;
      }
       
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
       //console.log({nonce});
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, shardsize, archonSPs/*, username*/).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: correctedPmt})
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
           testParams.testsPassed++; 
           console.log("passed proposeUpload testRequireArchonSPIsRegistered");
         })
         .on('receipt', (receipt) => {
           //console.log(receipt)
         })
         .catch(err => { 
           //console.error(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 /*|| err.toString().indexOf('revert archonSPs must be registered') > -1*/) {
            setTimeout(() => {testRequireArchonSPIsRegistered()}, 100);
           } else if (err.toString().indexOf('archonSPs must be registered') > -1) {
             //console.log(err);
             testParams.testsFailed++; 
             console.log("failed proposeUpload testRequireArchonSPIsRegistered");
           }
         });
         });
      });
     }

     var testRequireArchonSPIsRegisteredFalsePositive = function() {
         // test require(spAddress2Profile[archonSPs[i]].params != 0, 'archonSPs must be registered');
      var falsePositiveArchonSPs = archonSPs.slice();
      falsePositiveArchonSPs.push(wallets[9].address);
      var nonTrivialPmt = 1000000000000000;
      var correctedPmt = nonTrivialPmt;
      var r = correctedPmt % falsePositiveArchonSPs.length;
      if (r != 0) {
        correctedPmt += falsePositiveArchonSPs.length - r;
      }
       
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
       //console.log({nonce});
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, shardsize, falsePositiveArchonSPs/*, username*/).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: correctedPmt})
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
         })
         .on('receipt', (receipt) => {
           //console.log(receipt)
           testParams.testsFailed++; 
           console.log("failed proposeUpload testRequireArchonSPIsRegistered (false positive)");
         })
         .catch(err => { 
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 /*|| err.toString().indexOf('revert archonSPs must be registered') > -1*/) {
            setTimeout(() => {testRequireArchonSPIsRegisteredFalsePositive()}, 100);
           } else if (err.toString().indexOf('archonSPs must be registered') > -1) {
             //console.log(err);
             testParams.testsPassed++; 
             console.log("passed proposeUpload testRequireArchonSPIsRegistered (false positive)");
           }
         });
         });
      });
     }

     var testRequireShardsizeNontrivial = function() {
      var zeroShardsize = 0;
      var nonTrivialPmt = 1000000000000000;
      var correctedPmt = nonTrivialPmt;
      var r = correctedPmt % archonSPs.length;
      if (r != 0) {
        correctedPmt += archonSPs.length - r;
      }
       
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
       //console.log({nonce});
       var encoded = myContract.methods.proposeUpload(hashedArchonFilepath, containerSignatureR, containerSignatureS, params, zeroShardsize, archonSPs/*, username*/).encodeABI();
       wallet.signTransaction({from: wallet.address, to: contractAddress, gas: gas, nonce: nonce, data: encoded, value: correctedPmt})
         .then(ret => {
         //console.log({ret});
         web3.eth.sendSignedTransaction(ret.rawTransaction)
         .on('transactionHash', (transactionHash) => {
           //console.log({transactionHash});
           testParams.testsFailed++; 
           console.log("failed proposeUpload require shardsize nontrivial");
           testPaymentGranularityFinished = true;
         })
         .on('receipt', (receipt) => {
           //console.log(receipt)
         })
         .catch(err => { 
           //console.log(err);
           if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 /*|| err.toString().indexOf('revert archonSPs must be registered') > -1*/) {
            setTimeout(() => {testRequireShardsizeNontrivial()}, 100);
           } else if (err.toString().indexOf('shardsize must be non-trivial') > -1) {
             //console.log(err);
             testParams.testsPassed++; 
             console.log("passed proposeUpload require shardsize nontrivial");
           }
         });
         });
      })
      .catch(err => {
        if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1 /*|| err.toString().indexOf('revert archonSPs must be registered') > -1*/) {
          setTimeout(() => {testPaymentGranularity()}, 100);
        }
      });
     }

     var testPaymentAccounted = function() {
      var nonTrivialPmt = 1000000000000000;
      var numOfArchonSPsForThisTx = 3;
      var expectedPmt = 2 * Math.ceil(nonTrivialPmt / numOfArchonSPsForThisTx);
      // 2 because of two uploads in this suite
      myContract.methods.spAddress2SPProfile(archonSPs[2]).call()  // 2
      .then( res => {
        if (parseInt(res.earnings) === expectedPmt) {
          testParams.testsPassed++; 
          console.log("passed testPayment to sp is accounted for");
        } else {
          testParams.testsFailed++; 
          console.error("failed testPayment to sp is accounted for");
        }
      });
     }
  }
}

  return new TestProposeUpload;
}; // module.exports
