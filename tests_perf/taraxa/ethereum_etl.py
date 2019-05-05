import json
from multiprocessing import cpu_count
from operator import setitem
from blockchainetl.jobs.exporters.composite_item_exporter import CompositeItemExporter
from ethereumetl.executors.batch_work_executor import BatchWorkExecutor, RETRY_EXCEPTIONS
from ethereumetl.jobs.export_blocks_job import ExportBlocksJob
from ethereumetl.utils import rpc_response_batch_to_results
from ethereumetl.mappers.block_mapper import EthBlockMapper
from ethereumetl.providers.auto import get_provider_from_uri, DEFAULT_TIMEOUT
from ethereumetl.thread_local_proxy import ThreadLocalProxy
from ethereumetl.json_rpc_requests import generate_json_rpc, generate_get_block_by_number_json_rpc
from ethereumetl.utils import hex_to_dec


class _SimpleExporter(CompositeItemExporter):

    def __init__(self, callback):
        self.callback = callback

    def open(self):
        pass

    def close(self):
        pass

    def export_item(self, item):
        return self.callback(item)


class EmptyResponseException(Exception):
    pass


class _ExportBlocksJob(ExportBlocksJob):

    def _export_batch(self, block_number_batch):
        blocks_rpc = list(generate_get_block_by_number_json_rpc(block_number_batch, self.export_transactions))
        response = self.batch_web3_provider.make_batch_request(json.dumps(blocks_rpc))
        results = rpc_response_batch_to_results(response)
        blocks = []
        rpc_requests = []
        rpc_callbacks = []
        for block in results:
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
            batch_size = self.batch_work_executor.batch_size
            current_request_id = 0
            for batch_num in range(0, len(rpc_requests), batch_size):
                batch_request = json.dumps(rpc_requests[batch_num:batch_size])
                rpc_responses = self.batch_web3_provider.make_batch_request(batch_request)
                for response in rpc_responses:
                    assert current_request_id == response['id']
                    result = response.get('result')
                    if result is None:
                        raise EmptyResponseException(json.dumps(response))
                    rpc_callbacks[current_request_id](result)
                    current_request_id += 1
        for block in blocks:
            self.item_exporter.export_item(block)


def export_blocks_batch(start_block, end_block, on_block,
                        parallelism_factor=2.0,
                        batch_size=5000,
                        timeout=DEFAULT_TIMEOUT,
                        provider_uri='https://mainnet.infura.io'):
    assert batch_size > 0
    max_workers = int(cpu_count() * parallelism_factor)
    job = _ExportBlocksJob(
        start_block=start_block,
        end_block=end_block,
        batch_size=batch_size,
        batch_web3_provider=ThreadLocalProxy(lambda: get_provider_from_uri(provider_uri, timeout=timeout, batch=True)),
        max_workers=max_workers,
        item_exporter=_SimpleExporter(on_block),
        export_blocks=True,
        export_transactions=True)
    job.batch_work_executor = BatchWorkExecutor(batch_size, max_workers,
                                                retry_exceptions=(*RETRY_EXCEPTIONS, EmptyResponseException))
    job.run()
