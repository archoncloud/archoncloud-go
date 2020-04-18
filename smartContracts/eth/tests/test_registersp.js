// 5 tests
module.exports = () => {
var TestRegisterSP = function () {}
var zeroPad = function(input, padLength) {
  var zero = '0';
  var ret = input;
  while (ret.length < padLength) {
    ret = zero + ret;
  }
  return ret;
}

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
    //
    // populate ArchonSPProfile (under correct conditions)
    // populate url2Address (under correct conditions)
    // require guards
    //   alreadyRegistered
    //   non-trivial params 
    //   non-trivial nodeID 
    //   proper payment
    //
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
    var minAskPrice = Math.floor(Math.random() * Number.MAX_SAFE_INTEGER);
    var encMinAskPrice = zeroPad(minAskPrice.toString('16'), 16);
    // 4 country code
    var cc0 = 233;
    var cc1 = 1;
    var cc = zeroPad(cc0.toString('16'), 2) + zeroPad(cc1.toString('16'), 2);

    var params = encSlaLevel + encAvailableStorage + encBandwidth + encMinAskPrice + cc + "0000000000";
    
    params = params.replace("0x", "");
    params = Buffer.from(params, 'hex');
    var rand = Math.floor(Math.random() * 100000000);
    var nodeID = web3.utils.sha3("some/nodeID" + wallet.address + rand.toString());
    nodeID = nodeID.replace("0x", "");
    nodeID = Buffer.from(nodeID, 'hex');
    var hardwareProof = web3.utils.sha3("some preimage of hardwareProof");
    hardwareProof = hardwareProof.replace("0x", "");
    hardwareProof = Buffer.from(hardwareProof, 'hex');

    var goodPmt = 1000000000000000;
    var badPmt = 999999999999999;

    var initialRegisterSP = function() {
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
        /*myContract.methods.registerSP(params).estimateGas()
        .then(est => {
          console.log(est);*/
          var encoded = myContract.methods.registerSP(params, nodeID, hardwareProof).encodeABI();
          wallet.signTransaction({from:wallet.address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded, value: goodPmt})// here
            .then(ret => {
            //console.log({ret});
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('transactionHash', (transactionHash) => {
              //console.log({transactionHash});
              runTestSuite(); 
            })
            .on('receipt', (receipt) => {
              console.log(receipt.logs)
              //console.log(receipt)
            })
            .catch(err => { 
              console.log(err);
              if (err.toString().indexOf("correct nonce") > -1) {
                setTimeout(() => {initialRegisterSP()}, 100);
              }
            });
            });
        //});
      })
      .catch(err => { 
        console.log(err);
        if (err.toString().indexOf("correct nonce") > -1) {
          setTimeout(() => {initialRegisterSP()}, 100);
        }
      });
    }
    initialRegisterSP();

    function runTestSuite() {
      // testing storage start

      // populate ArchonSPProfile (under correct conditions)
      
      myContract.methods.spAddress2SPProfile(wallet.address).call()
      .then(res => {
        if (res.params === "0x" + Buffer.from(params).toString('hex') 
            && res.nodeID === "0x" + Buffer.from(nodeID).toString('hex')
            && parseInt(res.stake) === goodPmt
            && parseInt(res.earnings) === 0) {
          testParams.testsPassed++; 
          console.log("passed spAddress2SPProfile");
        } else {
          testParams.testsFailed++;
          console.log("failed spAddress2SPProfile");
        }
      });


      myContract.methods.nodeID2Address(nodeID).call()
      .then(res => {
        if (res === wallet.address) {
          testParams.testsPassed++; 
          console.log("passed url2Address");
        } else {
          testParams.testsFailed++;
          console.log("failed url2Address");
        }
      });
    
      // testing bad payment
      var testingBadPayment = function() { // testingRequire
        var rand = Math.floor(Math.random() * 100000000);
        var nodeID = web3.utils.sha3("some/nodeID" + testParams.wallets[1].address + rand.toString());
        nodeID = nodeID.replace("0x", "");
        nodeID = Buffer.from(nodeID, 'hex');
        web3.eth.getTransactionCount(testParams.wallets[1].address, 'pending')
        .then(nonce => {
          var encoded = myContract.methods.registerSP(params, nodeID, hardwareProof).encodeABI();
          testParams.wallets[1].signTransaction({from:testParams.wallets[1].address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded, value: badPmt})// here
            .then(ret => {
            //console.log({ret});
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('error', (error) => {
              //debug
              if (error.toString().indexOf("insufficient registerSP payment") > -1) {
                testParams.testsPassed++; 
                console.log("passed registerSP bad payment test");
                testingBadPaymentComplete = true;
              } else {
                //setTimeout(() => {testingBadPayment()}, 100);
              }
              })
            .on('transactionHash', (transactionHash) => {
              testParams.testsFailed++; 
              console.log("failed registerSP bad payment test");
            })
            .on('receipt', (receipt) => {})
            .catch(err => { 
                if (err.toString().indexOf("correct nonce") > -1) {
                  setTimeout(() => {testingBadPayment()}, 100);
                }
            });
            });
        })
        .catch(err => { 
            if (err.toString().indexOf("correct nonce") > -1) {
              setTimeout(() => {testingBadPayment()}, 100);
            }
        });
      }
      testingBadPayment();
      /// TODO UPDATE WITH PROPOSED UTILITY
      
      // testing already registered catch 
      var testingAlreadyRegistered = function() { // testingRequire
        var rand = Math.floor(Math.random() * 100000000);
        var nodeID = web3.utils.sha3("some/nodeID" + wallet.address + rand.toString());
        nodeID = nodeID.replace("0x", "");
        nodeID = Buffer.from(nodeID, 'hex');
        web3.eth.getTransactionCount(wallet.address, 'pending')
        .then(nonce => {
          var encoded = myContract.methods.registerSP(params, nodeID, hardwareProof).encodeABI();
          wallet.signTransaction({from:wallet.address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded, value: goodPmt})
            .then(ret => {
            //console.log({ret});
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('error', (error) => {
              //console.log(error)
              //debug
              if (error.toString().indexOf("this address is already a registeredSP") > -1) {
                console.log("debug error: ", error);
                testParams.testsPassed++; 
                console.log("passed registerSP already registered test");
              }
              })
            .on('transactionHash', (transactionHash) => {
              testParams.testsFailed++; 
              console.log("failed registerSP already registered test");
            })
            .on('receipt', (receipt) => {})
            .catch(err => { 
              if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
                setTimeout(() => {testingAlreadyRegistered()}, 100);
              }
            });
            });
        })
        .catch(err => { 
          if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
            setTimeout(() => {testingAlreadyRegistered()}, 100);
          }
        });
      }
      testingAlreadyRegistered();

      var testingTrivialParams = function() { // testingRequire
        var rand = Math.floor(Math.random() * 100000000);
        var nodeID = web3.utils.sha3("some/nodeID" + testParams.wallets[2].address + rand.toString());
        nodeID = nodeID.replace("0x", "");
        nodeID = Buffer.from(nodeID, 'hex');
        var trivialParams = Buffer.from("0000000000000000000000000000000000000000000000000000000000000000", 'hex')
        web3.eth.getTransactionCount(testParams.wallets[2].address, 'pending')
        .then(nonce => {
          var encoded = myContract.methods.registerSP(trivialParams, nodeID, hardwareProof).encodeABI();
          testParams.wallets[2].signTransaction({from:testParams.wallets[2].address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded, value: goodPmt})
            .then(ret => {
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('error', (error) => {
              //debug
              if (error.toString().indexOf("params must be nontrivial") > -1) {
                testParams.testsPassed++; 
                console.log("passed registerSP testingTrivialParams");
              } else if (error.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
                //setTimeout(() => {testingTrivialParams()}, 100);
              }
              })
            .on('transactionHash', (transactionHash) => {
              //console.log(transactionHash);
              testParams.testsFailed++; 
              console.log("failed registerSP testingTrivialParams");
            })
            .on('receipt', (receipt) => {
              //console.log(receipt);
            })
            .catch(err => { 
              if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
                setTimeout(() => {testingTrivialParams()}, 100);
              }
            });
            });
        })
        .catch(err => { 
          if (err.toString().indexOf("the tx doesn't have the correct nonce") > -1) {
            setTimeout(() => {testingTrivialParams()}, 100);
          }
        });
      }
      testingTrivialParams();

      // TODO  non-trivial nodeID //
    }
  }
};
  return new TestRegisterSP;
};// module.exports
