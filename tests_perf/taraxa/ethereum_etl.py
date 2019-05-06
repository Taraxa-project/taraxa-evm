import json
from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor
from operator import setitem

from ethereumetl.executors.batch_work_executor import RETRY_EXCEPTIONS
from ethereumetl.json_rpc_requests import generate_json_rpc, generate_get_block_by_number_json_rpc
from ethereumetl.providers.auto import get_provider_from_uri, DEFAULT_TIMEOUT
from ethereumetl.utils import dynamic_batch_iterator
from ethereumetl.utils import hex_to_dec
from requests.exceptions import ReadTimeout, Timeout, ConnectTimeout, ConnectionError
from os import cpu_count


class EmptyResponseException(Exception):
    pass


def _make_batch_request(batch_client, batch_request):
    results = []
    limit = len(batch_request)
    while len(results) < len(batch_request):
        next_portion = batch_request[len(results):limit]
        if not next_portion:
            break
        try:
            batch_response = batch_client.make_batch_request(json.dumps(next_portion))
            for response in batch_response:
                result = response.get('result')
                if result is None:
                    raise EmptyResponseException(response)
                results.append((result, response['id']))
            limit = len(batch_request) - len(results)
        except (*RETRY_EXCEPTIONS, ReadTimeout, EmptyResponseException) as e:
            print(f"Retrying on exception {e}")
            if isinstance(e, (ReadTimeout, EmptyResponseException, Timeout, ConnectionError, ConnectTimeout)):
                limit = max(10, int(limit / 2))
                print(f"Halved the batch size: {limit}")
    return results


def export_blocks_batch(block_number_batch, provider_uri, timeout=None):
    batch_web3_provider = get_provider_from_uri(provider_uri, timeout=timeout, batch=True)
    blocks_rpc = list(generate_get_block_by_number_json_rpc(block_number_batch, True))
    results = _make_batch_request(batch_web3_provider, blocks_rpc)
    blocks = []
    rpc_requests = []
    rpc_callbacks = []
    for block, _ in results:
        block['number'] = hex_to_dec(block['number'])
        uncles = block['uncles']
        block['uncleBlocks'] = [None] * len(uncles)
        for i in range(len(uncles)):
            rpc_requests.append(generate_json_rpc(
                method='eth_getUncleByBlockHashAndIndex',
                params=[block['hash'], hex(i)],
                request_id=len(rpc_requests)
            ))
            rpc_callbacks.append(lambda result, i=i, block=block: setitem(block['uncleBlocks'], i, result))
        for tx in block['transactions']:
            tx['blockNumber'] = hex_to_dec(tx['blockNumber'])
            tx['transactionIndex'] = hex_to_dec(tx['transactionIndex'])
            rpc_requests.append(generate_json_rpc(
                method='eth_getTransactionReceipt',
                params=[tx['hash']],
                request_id=len(rpc_requests)
            ))
            rpc_callbacks.append(lambda result, tx=tx: setitem(tx, 'receipt', result))
        blocks.append(block)
    if rpc_requests:
        batch_size = int(len(blocks) * 3)
        current_request_id = 0
        request_count = len(rpc_requests)
        for batch_num in range(0, request_count, batch_size):
            batch_request = rpc_requests[batch_num:batch_num + batch_size]
            results = _make_batch_request(batch_web3_provider, batch_request)
            for result, req_id in results:
                if current_request_id != req_id:
                    raise RuntimeError(f'{current_request_id} != {req_id}, '
                                       f'request_size: {len(batch_request)}, response_size: {len(results)}, '
                                       f'all_requests_count: {len(rpc_requests)}')
                rpc_callbacks[current_request_id](result)
                current_request_id += 1
        if current_request_id != request_count:
            raise RuntimeError(f'boo {current_request_id}, {request_count}')
    return blocks


def export_blocks(start_block, end_block, on_block,
                  parallelism_factor=2.0,
                  batch_size=5000,
                  timeout=DEFAULT_TIMEOUT,
                  provider_uri='https://mainnet.infura.io'):
    assert batch_size > 0

    def on_batch_result(batch_future):
        for block in batch_future.result():
            on_block(block)

    executor = ThreadPoolExecutor(max_workers=parallelism_factor * cpu_count())
    for batch in dynamic_batch_iterator(range(start_block, end_block + 1), lambda: batch_size):
        (executor.submit(export_blocks_batch, block_number_batch=batch, provider_uri=provider_uri, timeout=timeout)
         .add_done_callback(on_batch_result))
    print("waiting")
    executor.shutdown(wait=True)
