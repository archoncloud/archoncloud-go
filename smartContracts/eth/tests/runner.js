const testContractConstruction = require(__dirname + "/test_contract_construction.js");
const testRegisterUsername = require(__dirname + "/test_register_username.js");
const testRegisterSP = require(__dirname + "/test_registersp.js");
const testRegisterSPParseParams = require(__dirname + "/test_registersp_parse_params.js");
const testUnregisterSP = require(__dirname + "/test_unregistersp.js");
const testProposeUpload = require(__dirname + "/test_propose_upload.js");
const testProposeUploadParseParams = require(__dirname + "/test_propose_upload_parse_params.js");
const testArchonSPWithdrawal = require(__dirname + "/test_archonsp_withdrawal.js");
const testSlash = require(__dirname + "/test_slash.js");

const testExample = require(__dirname + "/testExample.js");

module.exports = () => {
const testContractConstructionRunner = testContractConstruction();
const testRegisterUsernameRunner = testRegisterUsername();
const testRegisterSPRunner = testRegisterSP();
const testRegisterSPParseParamsRunner = testRegisterSPParseParams();
const testUnregisterSPRunner = testUnregisterSP();
const testProposeUploadRunner = testProposeUpload();
const testProposeUploadParseParamsRunner = testProposeUploadParseParams();
const testArchonSPWithdrawalRunner = testArchonSPWithdrawal();
const testSlashRunner = testSlash();

const testExampleRunner = testExample();

var TestRunner = function() {}

TestRunner.prototype.run = function(testParams) {
  console.log("running tests\n");
  
  testParams.testsPassed = 0;
  testParams.testsFailed = 0;
  
  process.on('exit', function() {
    console.log("\n");
    var testsPassed = testParams.testsPassed;
    var testsFailed = testParams.testsFailed;
    console.log("Standard tests result:\n");
    console.log({testsPassed, testsFailed});
    console.log("--------------------------------------------------");
  });
  
  testExample.run(testParams);

  testContractConstructionRunner.run(testParams); // 6
  testRegisterUsernameRunner.run(testParams); // 6
  testRegisterSPRunner.run(testParams); // 5
  testProposeUploadRunner.run(testParams); // 8 - 1disabled
  
  testUnregisterSPRunner.run(testParams); // 2
  
  testSlashRunner.run(testParams); // 1
  //testArchonSPWithdrawalRunner.run(testParams); // TODO rework this test
}

TestRunner.prototype.runExtra = function(testParams) {
  console.log("running Extra tests\n");

  testParams.testsPassed = 0;
  testParams.testsFailed = 0;
  
  process.on('exit', function() {
    console.log("\n");
    var testsPassed = testParams.testsPassed;
    var testsFailed = testParams.testsFailed;
    console.log("Extra tests result:\n");
    console.log({testsPassed, testsFailed});
    console.log("--------------------------------------------------");
  });

  testRegisterSPParseParamsRunner.run(testParams); // 1
  testProposeUploadParseParamsRunner.run(testParams); // 1
}
  
  return new TestRunner;
}; // module.exports
