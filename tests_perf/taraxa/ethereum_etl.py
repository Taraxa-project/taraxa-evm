from multiprocessing import cpu_count

from blockchainetl.atomic_counter import AtomicCounter
from blockchainetl.exporters import JsonLinesItemExporter
from ethereumetl.executors.batch_work_executor import BatchWorkExecutor, RETRY_EXCEPTIONS
from ethereumetl.jobs.export_blocks_job import ExportBlocksJob
from ethereumetl.thread_local_proxy import ThreadLocalProxy
from ethereumetl.providers.auto import get_provider_from_uri, DEFAULT_TIMEOUT
from blockchainetl.jobs.exporters.composite_item_exporter import CompositeItemExporter
from ethereumetl.jobs.exporters.blocks_and_transactions_item_exporter import \
    TRANSACTION_FIELDS_TO_EXPORT, \
    BLOCK_FIELDS_TO_EXPORT


class _SimpleDictExporter(JsonLinesItemExporter):

    def export_item(self, item):
        self.file(dict(self._get_serialized_fields(item)))


class _SimpleCompositeExporter(CompositeItemExporter):

    def open(self):
        for item_type, callback in self.filename_mapping.items():
            fields = self.field_mapping.get(item_type)
            self.exporter_mapping[item_type] = _SimpleDictExporter(callback, fields_to_export=fields)
            self.counter_mapping[item_type] = AtomicCounter()


class EmptyResponseException(Exception):
    pass


class _BatchExecutor(BatchWorkExecutor):

    def _fail_safe_execute(self, work_handler, batch):
        def work_handler_proxy(*args, **kwargs):
            try:
                return work_handler(*args, **kwargs)
            except ValueError as e:
                print(e)
                raise EmptyResponseException()

        return super(_BatchExecutor, self)._fail_safe_execute(work_handler_proxy, batch)


def export_blocks_and_transactions(start_block, end_block,
                                   on_block=None,
                                   on_transaction=None,
                                   batch_size=5000,
                                   parallelism_factor=2.0,
                                   timeout=DEFAULT_TIMEOUT,
                                   provider_uri='https://mainnet.infura.io'):
    max_workers = int(cpu_count() * parallelism_factor)
    job = ExportBlocksJob(
        start_block=start_block,
        end_block=end_block,
        batch_size=batch_size,
        batch_web3_provider=ThreadLocalProxy(lambda: get_provider_from_uri(provider_uri,
                                                                           timeout=timeout,
                                                                           batch=batch_size > 0)),
        max_workers=max_workers,
        item_exporter=_SimpleCompositeExporter(
            filename_mapping={
                'block': on_block,
                'transaction': on_transaction
            },
            field_mapping={
                'block': BLOCK_FIELDS_TO_EXPORT,
                'transaction': TRANSACTION_FIELDS_TO_EXPORT
            }
        ),
        export_blocks=on_block is not None,
        export_transactions=on_transaction is not None)
    job.batch_work_executor = _BatchExecutor(batch_size, max_workers,
                                             retry_exceptions=(EmptyResponseException, *RETRY_EXCEPTIONS))
    job.run()
