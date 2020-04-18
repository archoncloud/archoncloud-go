#!/usr/bin/bash

firstFlag="foo"
if [ $1 ]
then
  firstFlag=$1
else
  echo "first flag must be non-trivial"
  echo "  valid flags are: -tests, -infura, -ganorge"
  exit 1
fi

if [ $firstFlag == "-tests" ];
  then
   compiles contract with extra test logic
  cat Archon.sol | sed "s/\/\/\-\-TEST\-\-\/\/ //g" > test.sol
  cat test.sol
  solcjs -o target --overwrite --bin --abi test.sol Ownable.sol SafeMath.sol Math.sol
  if [ $? -eq 0 ];
    then
    rm test.sol
    node deployTools.js -extraTests
    else
    echo "Solidity compilation NOT successful"
    rm test.sol
    exit 1
  fi
fi

# compiles production contract
solcjs -o target --overwrite --bin --abi Archon.sol Ownable.sol SafeMath.sol Math.sol

if [ $? -eq 0 ];
  then
  node deployTools.js $1 
  else
  echo "Solidity compilation NOT successful"
  exit 1
fi
