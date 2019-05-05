import asyncio
import json

from web3 import *

loop = asyncio.get_event_loop()


class AsyncWebsocketProvider(WebsocketProvider):
    _loop = WebsocketProvider._loop = loop

    async def coro_make_request(self, request_data):
        await self.conn.ws.send(request_data)
        return json.loads(await self.conn.ws.recv())

    def make_request(self, method, params):
        return {
            'result': {
                'async': self.coro_make_request(self.encode_rpc_request(method, params))
            }
        }

    @staticmethod
    async def unwrap(result):
        response = await result['async']
        return response['result']


i = 0


async def on_block(async_block):
    global i
    block = await async_block
    print(i, block['number'])
    i += 1


async def foo():
    provider = AsyncWebsocketProvider('wss://mainnet.infura.io/ws/v3/6f560dcc0ff74bc0aa6596b7fe253573')
    web3 = Web3(provider)
    async with provider.conn:
        result = asyncio.sleep(0)
        for i in range(100000):
            async_result = on_block(AsyncWebsocketProvider.unwrap(web3.eth.getBlock(i, full_transactions=True)))
            result = asyncio.gather(result, async_result)
            await asyncio.sleep(0)
        await result


loop.run_until_complete(foo())
