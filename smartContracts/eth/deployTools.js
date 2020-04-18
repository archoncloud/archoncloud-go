const fs = require("fs");
const unitTests = require(__dirname + "/tests/runner.js");
const unitTestRunner = unitTests();
const ethers = require("ethers");
const Web3 = require("web3");
const infuraUrl = ""; // PUT INFURA URL HERE
const ganacheUrl = "http://127.0.0.1:7545";
const ganorgeUrl = "http://ethtest.archon.cloud";
var web3Url = "";

if (process.argv[2] === undefined) {
  console.error("need to specify -tests, or -infura, or ganorge");
  process.exit();
}

var mnemonic = "";

if (process.argv[2] === "-tests" || process.argv[2] === "-extraTests") {
  web3Url = ganacheUrl;
  mnemonic = "actor win cheese history trial seed wheel squeeze shoot genuine stock physical";
} else if (process.argv[2] === "-infura") {
  web3Url = infuraUrl;
  mnemonic = "toss fruit saddle spike across mountain stamp tooth office credit profit analyst";
} else if (process.argv[2] === "-ganorge") {
  web3Url = ganorgeUrl;
  mnemonic = "toss fruit saddle spike across mountain stamp tooth office credit profit analyst";
}
console.log({web3Url});
var web3 = new Web3(web3Url,
                      null,
                      {transactionConfirmationBlocks:1 /*critical to prevent hanging*/});

var wallet = ethers.Wallet.fromMnemonic(mnemonic);
var path = wallet.path;
wallet = web3.eth.accounts.privateKeyToAccount(wallet.privateKey);

/*var encryptedWallet = wallet.encrypt("ganacheTestingWallet1");
console.log(JSON.stringify(encryptedWallet));*/

var wallets = [wallet];
for (var i = 1; i < 10; i++) {
  var idx = i - 1;
  var reg = "/" + idx.toString() + "$/";
  var rep = i.toString();
  path = path.replace(eval(reg), rep);
  var newWallet = ethers.Wallet.fromMnemonic(mnemonic, path);
  wallets.push(web3.eth.accounts.privateKeyToAccount(newWallet.privateKey));
  var encryptedWallet = wallets[i].encrypt("ethTestingWallet");
  console.log(encryptedWallet);
  fs.writeFileSync(__dirname + "/newWallets/ethTestingWallet" + i + ".json", JSON.stringify(encryptedWallet));
}

let abi = JSON.parse(fs.readFileSync(__dirname + "/target/Archon_sol_Archon.abi"));
let bin = fs.readFileSync(__dirname + "/target/Archon_sol_Archon.bin");

let code = '0x' + bin;

let testAbi = JSON.parse(fs.readFileSync(__dirname + "/target/test_sol_Archon.abi"));
let testBin = fs.readFileSync(__dirname + "/target/test_sol_Archon.bin");
let testCode = '0x' + testBin;

var deployToGanacheAndTest = (testType) => {
    var testParams = {
      abi: abi,
      code: code,
      wallets: wallets,
      web3: web3
    };
    if (testType === "-extraTests") {
      testParams.abi = testAbi;
      testParams.code = testCode;
      unitTestRunner.runExtra(testParams);
    } else {
      // standard tests of deployement contract
      unitTestRunner.run(testParams);
    }
}

var deployToExternalTestnet = () => {
  // DEPLOYING CONTRACT
  
  wallet = wallets[9]; // FIXME note that owner is wallet 9

  /*web3.eth.getBalance(wallet.address)
  .then(res => {
    console.log(res);
  }); // SANITY CHECK*/
  console.log("Deploying the contract");
  console.log("debug wallet ", wallet.address);
  web3.eth.getTransactionCount(wallet.address, 'pending') 
  .then(nonce => {
    web3.eth.estimateGas(
      {from: wallet.address, nonce: nonce, data: code}
    )
    .then( est => {
      fee = est + 10000;
      console.log({fee});
      wallet.signTransaction({from:wallet.address, gas: fee, nonce: nonce, data: code})
      .then(ret => {
        web3.eth.sendSignedTransaction(ret.rawTransaction)
        .on('error', (error) => {console.log(error)})
        .on('transactionHash', (transactionHash) => {
        
	})
        .on('receipt', (receipt) => {
          console.log({receipt});
        })
        .catch(err => console.error(err));
      });
    })
    .catch(err => console.error(err));
  });
}


if (process.argv[2] === "-tests") {
  deployToGanacheAndTest(process.argv[2]);
} else if (process.argv[2] === "-extraTests") {
  deployToGanacheAndTest(process.argv[2]);
} else if (process.argv[2] === "-infura") {
  deployToExternalTestnet(); 
} else if (process.argv[2] === "-ganorge") {
  /*console.log(wallets[0].address);
  web3.eth.getBalance(wallets[0].address)
  .then(res => {
    console.log(res);
  }); // SANITY CHECK*/

  deployToExternalTestnet();
}
