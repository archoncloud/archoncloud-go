// 8 tests

module.exports = () => {
var TestConstruction = function () {}

TestConstruction.prototype.run = function (testParams) {
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
          .on('error', (error) => {console.log(error)})
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
    // TEST PROPER CONSTRUCTIONS
    myContract.methods.registerCost().call()
    .then( res => {
      if (parseInt(res) === 100000000000) {
        testParams.testsPassed++; 
        console.log("passed registerCost constructions");
      } else {
        testParams.testsFailed++; 
        console.error("failed registerCost constructions");
      }
    });

    myContract.methods.registerCostScalar().call()
    .then( res => {
      if (parseInt(res) === 10000) {
        testParams.testsPassed++; 
        console.log("passed registerCostScalar constructions");
      } else {
        testParams.testsFailed++; 
        console.error("failed registerCostScalar constructions");
      }
    });

    myContract.methods.slaLevelScalar().call()
    .then( res => {
      if (parseInt(res) === 0) {
        testParams.testsPassed++; 
        console.log("passed slaLevelScalar constructions");
      } else {
        testParams.testsFailed++; 
        console.error("failed slaLevelScalar constructions");
      }
    });

    myContract.methods.availableStorageScalar().call()
    .then( res => {
      if (parseInt(res) === 0) {
        testParams.testsPassed++; 
        console.log("passed availableStorageScalar constructions");
      } else {
        testParams.testsFailed++; 
        console.error("failed availableStorageScalar constructions");
      }
    });

    myContract.methods.bandwidthScalar().call()
    .then( res => {
      if (parseInt(res) === 0) {
        testParams.testsPassed++; 
        console.log("passed bandwidthScalar constructions");
      } else {
        testParams.testsFailed++; 
        console.error("failed bandwidthScalar constructions");
      }
    });
    
    myContract.methods.minAskPriceScalar().call()
    .then( res => {
      if (parseInt(res) === 0) {
        testParams.testsPassed++; 
        console.log("passed minAskPriceScalar constructions");
      } else {
        testParams.testsFailed++; 
        console.error("failed minAskPriceScalar constructions");
      }
    });
    
  }
}

  return new TestConstruction;
}; // module.exports
