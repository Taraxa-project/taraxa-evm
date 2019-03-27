const solc = require('solc')
const {resolve, relative, dirname} = require('path')
const glob = require('glob')
const {readFileSync, writeFileSync} = require('fs')
const {assign} = Object

const THIS_DIR = dirname(__filename)
const CONTRACTS_SRC = resolve(THIS_DIR, 'src')
const SOLC_OUT_PATH = resolve(THIS_DIR, 'solc_out.json')

const cli = {
    compile: () => {
        const output = solc.compile(JSON.stringify({
            language: "Solidity",
            sources: glob.sync(resolve(CONTRACTS_SRC, '**/*.sol'))
                .reduce((result, contractPath) => assign(result, {
                    [relative(CONTRACTS_SRC, contractPath)]: {
                        content: readFileSync(contractPath, {
                            encoding: 'utf8'
                        })
                    }
                }), {}),
            settings: {
                evmVersion: "constantinople",
                outputSelection: {
                    "*": {
                        "*": ["*"]
                    }
                }
            }
        }))
        writeFileSync(SOLC_OUT_PATH, output)
        process.stdout.write(output)
    },
    get_code: (contractFile, contractName) => process.stdout.write(
        solcOut(contractFile, contractName).evm.bytecode.object
    ),
    generate_call: (contractFile, contractName, functionName, ...args) => process.stdout.write(
        require("web3-eth-abi")
            .AbiCoder()
            .encodeFunctionCall(
                solcOut(contractFile, contractName)
                    .abi.find(signature => signature.name === functionName),
                args
            )
    )

}

const [command, ...args] = process.argv.slice(2)
cli[command](...args)

function solcOut(contractFile, contractName) {
    const solcOut = JSON.parse(readFileSync(SOLC_OUT_PATH))
    return solcOut.contracts[contractFile][contractName]
}