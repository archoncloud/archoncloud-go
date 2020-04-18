// 2 tests
// // TODO UPDATE
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
  const contractAddress = testParams.contractAddress;
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
    // TEST UNREGISTERSP 
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
    var nodeID = web3.utils.sha3("some/upload/Url");
    nodeID = nodeID.replace("0x", "");
    nodeID = Buffer.from(nodeID, 'hex');
    var hardwareProof = web3.utils.sha3("some preimage of hardwareProof");
    hardwareProof = hardwareProof.replace("0x", "");
    hardwareProof = Buffer.from(hardwareProof, 'hex');
    var goodPmt = 1000000000000000;
    
    // first registersp
    var initialRegisterSP = function() {
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
        /*myContract.methods.registerSP(params).estimateGas()
        .then(est => {
          console.log(est);*/
          var encoded = myContract.methods.registerSP(params, nodeID, hardwareProof).encodeABI();
          wallet.signTransaction({from:wallet.address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded, value: goodPmt})// here
            .then(ret => {
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('transactionHash', (transactionHash) => {
              unregisterSP(); 
            })
            .on('receipt', (receipt) => {
              //console.log(receipt)
            })
            .catch(err => { 
              if (err.toString().indexOf("correct nonce") > -1) {
                setTimeout(() => {initialRegisterSP()}, 100);
              }
            });
            });
        //});
      })
      .catch(err => { 
        if (err.toString().indexOf("correct nonce") > -1) {
          setTimeout(() => {initialRegisterSP()}, 100);
        }
      });
    }
    initialRegisterSP();
    // then unregistersp
    
    var unregisterSP = () => {
      web3.eth.getTransactionCount(wallet.address, 'pending')
      .then(nonce => {
        /*myContract.methods.registerSP(params).estimateGas()
        .then(est => {
          console.log(est);*/
          var encoded = myContract.methods.unregisterSP().encodeABI();
          wallet.signTransaction({from:wallet.address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded})// here
            .then(ret => {
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('transactionHash', (transactionHash) => {
            })
            .on('receipt', (receipt) => {
              testSPProfile();
            })
            .catch(err => {
              // TODO FAIL IF REVERT
              if (err.toString().indexOf("correct nonce") > -1) {
                setTimeout(() => {unregisterSP()}, 100);
              } else if (err.toString().indexOf("revert") > -1) {
                testParams.testsFailed++;
                console.log("failed unregisterSP");
              }
            });
            });
        //});
      })
      .catch(err => {
        if (err.toString().indexOf("correct nonce") > -1) {
          setTimeout(() => {unregisterSP()}, 100);
        }
      });
    }

    var testSPProfile = () => {

      myContract.methods.spAddress2SPProfile(wallet.address).call()
      .then(res => {
        if (res.params === "0x0000000000000000000000000000000000000000000000000000000000000000" 
            && res.nodeID === "0x0000000000000000000000000000000000000000000000000000000000000000"
            && parseInt(res.stake) === 0 
            && parseInt(res.earnings) === 0) {
          testParams.testsPassed++; 
          console.log("passed unregisterSP spAddress2SPProfile");
        } else {
          testParams.testsFailed++;
          console.log("failed unregisterSP spAddress2SPProfile");
        }
      });


      myContract.methods.nodeID2Address(nodeID).call()
      .then(res => {
        if (res === "0x0000000000000000000000000000000000000000") {
          testParams.testsPassed++; 
          console.log("passed unregisterSP url2Address");
        } else {
          testParams.testsFailed++;
          console.log("failed unregisterSP url2Address");
        }
      });
    }
  }
};
  return new TestRegisterSP;
};// module.exports
