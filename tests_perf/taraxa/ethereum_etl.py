from multiprocessing import cpu_count

from .util import call


def export_blocks_and_transactions(start_block, end_block, blocks_out, transactions_out,
                                   batch_size=5000,
                                   worker_count=cpu_count(),
                                   provider_url='https://mainnet.infura.io'):
    call("ethereumetl export_blocks_and_transactions "
         f"--start-block {start_block} --end-block {end_block} "
         f"--batch-size {batch_size} "
         f"-w {worker_count} "
         f"--blocks-output {blocks_out} "
         f"--transactions-output {transactions_out} "
         f"--provider-uri {provider_url}")
