import asyncio
import json

from web3 import *


web3 = Web3(HTTPProvider(endpoint_uri='https://mainnet.infura.io/v3/6f560dcc0ff74bc0aa6596b7fe253573'))

b = web3.eth.getBlock(0, full_transactions=True)
print(b)