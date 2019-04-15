#!usr/bin/env python3

import sys
import os
from multiprocessing import cpu_count
from pathlib import Path

from ethereumetl.cli import cli as ethereum_etl

this_dir = Path(__file__).absolute().parent
etl_dir = this_dir.joinpath('ethereum_etl')

sys.path.append(str(this_dir))


def export(start_block, end_block, batch_size, worker_count):
    ethereum_etl("export_blocks_and_transactions "
                 f"--start-block {start_block} --end-block {end_block} "
                 f"--batch-size {batch_size} "
                 f"-w {worker_count} "
                 f"--blocks-output {etl_dir.joinpath('blocks.json')} "
                 f"--transactions-output {etl_dir.joinpath('transactions.json')} "
                 "--provider-uri https://mainnet.infura.io"
                 .split(" "))


batch = 5000
last = 7519165
export(last - 100, last, batch, cpu_count())


from google.cloud import bigquery

bq = bigquery.Client.from_service_account_json(this_dir.joinpath("taraxa-perf-test-creds.json"))

query_job = bq.query(
    "SELECT * FROM `bigquery-public-data.ethereum_blockchain.transactions` "
    f'WHERE block_number > {last - 100}',
    location="US",
)

for row in query_job:
    print('--------------')
    for k, v in row.items():
        print(f'{k}:{v}')