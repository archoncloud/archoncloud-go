// 6 tests
module.exports = () => {
var TestConstruction = function () {}

TestConstruction.prototype.run = function (testParams) {
  const abi = testParams.abi;
  const code = testParams.code;
  var wallets = testParams.wallets;
  var wallet = wallets[0];
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
    // for now register all addresses
    
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
              testRegisteredUsername(i);
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
    registerUsername(0);

    /*var indexArray = [...Array(wallets.length).keys()];
    indexArray.forEach((i) => {
      registerUsername(i);
    });*/

    var testRegisteredUsername = (i) => {
      var username = web3.utils.sha3("generating random username" + i);
      username = username.replace("0x", "");
      username = Buffer.from(username, 'hex');
      var publicKeyX = web3.utils.sha3("generating fake publicKeyX" + i);
      publicKeyX = publicKeyX.replace("0x", "");
      publicKeyX = Buffer.from(publicKeyX, 'hex');
      var publicKeyY = web3.utils.sha3("generating fake publicKeyY" + i); // TODO MAKE REAL LATERS
      publicKeyY = publicKeyY.replace("0x", "");
      publicKeyY = Buffer.from(publicKeyY, 'hex');
      myContract.methods.username2UserProfile(username).call()
      .then(res => {
        if (res.usersPublicKeyX === "0x" + Buffer.from(publicKeyX).toString('hex')
            && res.usersPublicKeyY === "0x" + Buffer.from(publicKeyY).toString('hex') 
            && res.usersAddress === wallets[i].address) {
          testParams.testsPassed++; 
          console.log("passed testRegistedUsername username2UserProfile");
        } else {
          testParams.testsFailed++;
          console.log("failed testRegistedUsername username2UserProfile");
        }
      });

      myContract.methods.address2Username(wallets[i].address).call()
      .then(res => {
        if (res === "0x" + Buffer.from(username).toString('hex')) {
          testParams.testsPassed++; 
          console.log("passed testRegistedUsername address2Username");
        } else {
          testParams.testsFailed++;
          console.log("failed testRegistedUsername address2Username");
        }
      });
    }

    // TEST ALREADY REGISTERED CHECK
    var alreadyRegisteredCheckTest = (i) => {
      var username = web3.utils.sha3("generating random username" + i);
      username = username.replace("0x", "");
      username = Buffer.from(username, 'hex');
      var publicKeyX = web3.utils.sha3("generating fake publicKeyX" + i);
      publicKeyX = publicKeyX.replace("0x", "");
      publicKeyX = Buffer.from(publicKeyX, 'hex');
      var publicKeyY = web3.utils.sha3("generating fake publicKeyY" + i); // TODO MAKE REAL LATERS
      publicKeyY = publicKeyY.replace("0x", "");
      publicKeyY = Buffer.from(publicKeyY, 'hex');
      var testAlreadyRegistered = (i) => {
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
                // FAILURE
                testParams.testsFailed++; 
                console.log("failed testRegistedUsername alreadyRegisteredCheckTest");
              })
              .catch(err => { 
                if (err.toString().indexOf("correct nonce") > -1) {
                  setTimeout(() => {registerUsername(i)}, 100);
                } else if (err.toString().indexOf("address already registered") > -1) {
                  // PASS 
                  testParams.testsPassed++; 
                  console.log("passed testRegistedUsername alreadyRegisteredCheckTest");
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
              testAlreadyRegistered(i);
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
    alreadyRegisteredCheckTest(1); 

    // TEST NON-TRIVIAL PARAMS CHECK
    var testTrivialParams = (i, paramNum) => {
      var zeros = "0000000000000000000000000000000000000000000000000000000000000000";
      var username = web3.utils.sha3("generating random username" + i);
      username = username.replace("0x", "");
      username = Buffer.from(username, 'hex');
      var publicKeyX = web3.utils.sha3("generating fake publicKeyX" + i);
      publicKeyX = publicKeyX.replace("0x", "");
      publicKeyX = Buffer.from(publicKeyX, 'hex');
      var publicKeyY = web3.utils.sha3("generating fake publicKeyY" + i); // TODO MAKE REAL LATERS
      publicKeyY = publicKeyY.replace("0x", "");
      publicKeyY = Buffer.from(publicKeyY, 'hex');

      switch (paramNum) {
        case 0:
          username = Buffer.from(zeros, 'hex');
          break;
        case 1:
          publicKeyX = Buffer.from(zeros, 'hex');
          break;
        case 2:
          publicKeyY = Buffer.from(zeros, 'hex');
      }

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
              // FAIL
              testParams.testsFailed++; 
              console.log("failed registerUsername testTrivialParams ", paramNum);
            })
            .catch(err => { 
              if (err.toString().indexOf("correct nonce") > -1) {
                setTimeout(() => {registerUsername(i)}, 100);
              } else if (err.toString().indexOf("must have non-trivial parameters") > -1) {
                // PASS
                testParams.testsPassed++; 
                console.log("passed registerUsername testTrivialParams ", paramNum);
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
    testTrivialParams(2, 0);
    testTrivialParams(3, 1);
    testTrivialParams(4, 2);
  }
}

  return new TestConstruction;
}; // module.exports
