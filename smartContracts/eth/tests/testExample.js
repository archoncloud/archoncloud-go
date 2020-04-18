// // TODO UPDATE
module.exports = () => {
/*var zeroPad = function(input, padLength) {
  var zero = '0';
  var ret = input;
  while (ret.length < padLength) {
    ret = zero + ret;
  }
  return ret;
}*/

var TestExample = function () {}
TestExample.prototype.run = function(testParams) {
  const abi = testParams.abi;
  const code = testParams.code;
  const contractAddress = testParams.contractAddress;
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


    //var params = encSlaLevel + encAvailableStorage + encBandwidth + encMinAskPrice + cc + "0000000000";

    //params = params.replace("0x", "");
    //params = Buffer.from(params, 'hex');
    var firstTest = function() {
      web3.eth.getTransactionCount(wallets[1].address, 'pending')
      .then(nonce => {
        /*myContract.methods.registerSP(params).estimateGas()
        .then(est => {
          console.log(est);*/
          var encoded = myContract.methods.ExampleFunction(param1, param2).encodeABI();
          wallets[1].signTransaction({from:wallets[1].address, to: contractAddress, gas: 6721974/*est*/, nonce: nonce, data: encoded, value: 0})// here
            .then(ret => {
            web3.eth.sendSignedTransaction(ret.rawTransaction)
            .on('transactionHash', (transactionHash) => {
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
    firstTest();
    

    var TESTKEY = web3.utils.sha3("some/upload/Url");
    TESTKEY = TESTKEY.replace("0x", "");
    TESTKEY = Buffer.from(TESTKEY, 'hex');

    /*myContract.methods.STORAGEVARIABLE(/*TESTKEY*//*).call()
    .then( res => {
      if (parseInt(res) === 0) {
        testParams.testsPassed++; 
        console.log("passed minAskPriceScalar constructions");
      } else {
        testParams.testsFailed++; 
        console.error("failed minAskPriceScalar constructions");
      }
    });*/

  }
};
  return new TestExample;
};// module.exports
