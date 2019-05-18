import json
from pathlib import Path
from tempfile import gettempdir
from zipfile import ZipFile, ZIP_DEFLATED

from apps.blockchain_data import BlockDB
from apps.compute_state_db import map_block
from taraxa import rocksdb_util
from taraxa.lib_taraxa_evm import LibTaraxaEvm
from taraxa.util import raise_if_not_none
from itertools import combinations
from os import cpu_count
from concurrent.futures import ProcessPoolExecutor
import statistics

intervals = [
    # (778483, 808909),
    # (1620940, 1657167),
    # (2912407, 2948852),
    # (3800776, 3831962),
    # (4652926, 4728185)
    # (4795000, 4797000)
    # (4795000, 4795815),
    # (4650000, 4650510),
    # (4650000, 4657000),
    # (4750000, 4750618),
    # (4750000, 4800000),
    (2176170, 2218406),
    (2640392, 2682710),
    (3106163, 3148470),
    (3565830, 3606478),
    (3980317, 4014251),
    (4296438, 4317536),
    (4687867, 4728185)
]

BLOCK_STEP = 10
warmup_rounds = 2
test_rounds = 5

BASE_DIR = Path('/mnt/xvdf/perf_test/')

PROJECT_NAME = 'test_mode_metrics_16'

BASE_DIR.mkdir(exist_ok=True, parents=True)

dummy_state_dir = BASE_DIR.joinpath(f'{PROJECT_NAME}_dummy_state')
dummy_state_dir.mkdir(exist_ok=True, parents=True)

block_db_conf = rocksdb_util.Config(read_only=True, path=str(BASE_DIR.joinpath('blocks')))
block_db = BlockDB(block_db_conf.new_db())


def new_vm(partition):
    print("building lib")
    library_path = Path(gettempdir()).joinpath(f'{PROJECT_NAME}_taraxa_vm_{partition}').joinpath('taraxa_vm.so')
    LibTaraxaEvm.build(library_path)
    lib_taraxa_vm = LibTaraxaEvm(library_path)
    print("built lib")

    conf = {
        'readDB': {
            'cacheSize': 3000,
            'db': {
                'type': 'rocksdb',
                'options': {
                    'file': str(BASE_DIR.joinpath('ethereum_emulated_state_rocksdb')),
                    'readOnly': True
                }
            }
        },
        'writeDB': {
            'cacheSize': 3000,
            'db': {
                'type': 'rocksdb',
                'options': {
                    'file': str(dummy_state_dir.joinpath(f'state_db_{partition}')),
                }
            }
        },
        'blockDB': {
            'type': 'rocksdb',
            'options': {
                'file': block_db_conf.path,
                'readOnly': True
            }
        },
        'conflictDetectorInboxPerTransaction': 500,
        'threadPool': {
            'threadCount': cpu_count() * 2,
            'queueSize': cpu_count() * 50
        }
    }
    taraxa_vm_handle, err = lib_taraxa_vm.call("NewVM", conf)
    raise_if_not_none(err, RuntimeError)
    print("built vm")
    return lib_taraxa_vm.as_ptr(taraxa_vm_handle)


partitions = 1
vm_instances = {}


def process_block(block_num):
    partition = block_num % partitions
    taraxa_vm_ptr = vm_instances.get(partition)
    if not taraxa_vm_ptr:
        taraxa_vm_ptr = new_vm(partition)
        vm_instances[partition] = taraxa_vm_ptr
    prev_block = block_db.get_block(block_num - 1)
    block = block_db.get_block(block_num)
    state_transition_request = {
        "stateRoot": prev_block['stateRoot'],
        "block": map_block(block),
    }
    all_tx_ids = list(range(len(block['transactions'])))
    conflicting_tx_ids = []
    schedule_metrics_samples = []
    # print('running schedule generation...')
    for round in range(test_rounds + warmup_rounds):
        schedule, schedule_metrics, err = taraxa_vm_ptr.call("GenerateSchedule", state_transition_request)
        raise_if_not_none(err)
        conflicting_tx_ids = schedule['sequential'] = schedule.get('sequential') or []
        if round >= warmup_rounds:
            schedule_metrics_samples.append(schedule_metrics)
    # print(f'running state transition configurations...')
    run_configurations = [
        ('read_only', {}),
        ('read_write', {'doCommits': True}),
        ('read_write_in_separate_db', {'doCommitsInSeparateDB': True}),
        ('read_write_commit_sync', {'commitSync': True})
    ]
    full_sequential_configurations = [(f'{name}_sequential', {'sequentialTx': all_tx_ids, **config})
                                      for name, config in run_configurations]
    taraxa_configurations = [(f'{name}_taraxa', {'sequentialTx': conflicting_tx_ids, **config})
                             for name, config in run_configurations]
    run_configurations.extend(full_sequential_configurations)
    run_configurations.extend(taraxa_configurations)
    metrics_by_config = {}
    for config_name, config in run_configurations:
        # print(f"running configuration: {config_name}: {config}")
        # print(f'warming up ({warmup_rounds} rounds)... ')
        for _ in range(warmup_rounds):
            taraxa_vm_ptr.call("TestMode", state_transition_request, config)
        # print(f'testing ({test_rounds}) roudnds...')
        for _ in range(test_rounds):
            [config_metrics] = taraxa_vm_ptr.call("TestMode", state_transition_request, config)
            metrics_by_config.setdefault(config_name, []).append(config_metrics)
    return {
        'blockNumber': block_num,
        'txCount': len(block['transactions']),
        'conflictingTx': conflicting_tx_ids,
        'metrics': metrics_by_config,
        'scheduleMetrics': schedule_metrics_samples
    }


results_db = rocksdb_util.Config(path=str(BASE_DIR.joinpath(PROJECT_NAME)), opts={'create_if_missing': True}).new_db()
total = sum(end - start + 1 for start, end in intervals)
processed = 0
for start, end in intervals:
    current_block = start
    while current_block <= end:
        if not results_db.get(BlockDB.block_key_encode(current_block)):
            print(f'processing block {current_block}, progress: {processed / total}%')
            result = process_block(current_block)
            results_db.put(BlockDB.block_key_encode(current_block), json.dumps(result).encode())
        current_block += BLOCK_STEP
        processed += BLOCK_STEP

import numpy


def print_summary_stats(name, array):
    print(f'min {name}: {numpy.percentile(array, 0)}\n'
          f'pct. 25 {name}: {numpy.percentile(array, 25)}\n'
          f'median {name}: {numpy.percentile(array, 50)}\n'
          f'pct. 75 {name}: {numpy.percentile(array, 75)}\n'
          f'max {name}: {numpy.percentile(array, 100)}\n'
          f'mean {name}: {numpy.mean(array)}\n'
          f'std. dev {name}: {numpy.std(array)}\n')


def deep_avg_array(metrics_samples, key):
    tx_metric_values_by_index = {}
    for metrics in metrics_samples:
        for i, tx_metrics in enumerate(metrics[key]):
            for metric, value in tx_metrics.items():
                tx_metric_values_by_index.setdefault(i, {}).setdefault(metric, []).append(value)
    transaction_metrics_mean = [None] * len(tx_metric_values_by_index)
    for i, tx_metric_values in tx_metric_values_by_index.items():
        transaction_metrics_mean[i] = {k: numpy.mean(v) for k, v in tx_metric_values.items()}
    return transaction_metrics_mean


def normalize_result(result):
    mean_metrics_by_config = result['mean_metrics'] = {}
    for config_name, metrics_samples in result['metrics'].items():
        metrics_mean = mean_metrics_by_config[config_name] = {}
        for metric_group in ('main', 'committer'):
            values_by_metric = {}
            for metrics in metrics_samples:
                for metric, value in metrics[metric_group].items():
                    values_by_metric.setdefault(metric, []).append(value)
            metrics_mean[metric_group] = {k: numpy.mean(v) for k, v in values_by_metric.items()}
        metrics_mean['transactions'] = deep_avg_array(metrics_samples, 'transactions')
        metrics_mean['transactions_mean'] = {
            metric: numpy.mean([metrics_mean[metric] for metrics_mean in metrics_mean['transactions']] or [0])
            for metric in ['totalTime', 'localCommit', 'createDB']
        }
    metrics_samples = result['scheduleMetrics']
    schedule_mean = mean_metrics_by_config['schedule_generation'] = {
        'main': {
            'totalTime': numpy.mean([s['totalTime'] for s in metrics_samples]),
            'transactionsSync': 0,
            'commitsSync': 0,
        },
        'transactions': deep_avg_array(metrics_samples, 'transactionMetrics'),
    }
    result['transactions_mean'] = {
        metric: numpy.mean([metrics_mean[metric] for metrics_mean in schedule_mean['transactions']] or [0])
        for metric in ['totalTime']
    }
    return result


from shutil import rmtree

results_db_path = BASE_DIR.joinpath(f'{PROJECT_NAME}_normalized')
# rmtree(results_db_path, ignore_errors=True)
results_norm_db = rocksdb_util.Config(path=str(results_db_path), opts={'create_if_missing': True}).new_db()

results = {}


def on_result(result):
    results[result['blockNumber']] = result
    print(f'normalizing metrics progress: {len(results) * BLOCK_STEP / total}%')


executor = ProcessPoolExecutor(max_workers=int(cpu_count() * 1.5))

itr = results_db.iteritems()
itr.seek_to_first()
for k, v in itr:
    from_db = results_norm_db.get(k)
    if from_db:
        on_result(json.loads(from_db))
    else:

        def cb(future):
            result = future.result()
            results_norm_db.put(k, json.dumps(result).encode())
            on_result(result)


        executor.submit(normalize_result, json.loads(v)).add_done_callback(cb)
executor.shutdown(wait=True)
# for v in itr:
#     result = json.loads(v)
#     mean_metrics_by_config = result['mean_metrics'] = {}
#     for config_name, metrics_samples in result['metrics'].items():
#         if config_name not in filtered_configs:
#             continue
#         metrics_mean = mean_metrics_by_config[config_name] = {}
#         for metric_group in ('main', 'committer'):
#             values_by_metric = {}
#             for metrics in metrics_samples:
#                 for metric, value in metrics[metric_group].items():
#                     values_by_metric.setdefault(metric, []).append(value)
#             metrics_mean[metric_group] = {k: numpy.mean(v) for k, v in values_by_metric.items()}
#         tx_metric_values_by_index = {}
#         for metrics in metrics_samples:
#             for i, tx_metrics in enumerate(metrics['transactions']):
#                 for metric, value in tx_metrics.items():
#                     tx_metric_values_by_index.setdefault(i, {}).setdefault(metric, []).append(value)
#         transaction_metrics_mean = [None] * len(tx_metric_values_by_index)
#         for i, tx_metric_values in tx_metric_values_by_index.items():
#             transaction_metrics_mean[i] = {k: numpy.mean(v) for k, v in tx_metric_values.items()}
#         metrics_mean['transactions'] = {
#             f'mean_{metric}': numpy.mean([metrics_mean[metric] for metrics_mean in transaction_metrics_mean] or [0])
#             for metric in ('totalTime', 'localCommit', 'createDB')
#         }
#     results.append(result)
#     print(result['blockNumber'])


filtered_configs = {
    'read_only', 'read_only_sequential', 'read_only_taraxa',
    'schedule_generation',
    'read_write_in_separate_db',
    'read_write_in_separate_db_taraxa',
    'read_write_commit_sync_sequential'
}
comparisons = {}
total_time_fractions = {}
for i, result in results.items():
    mean_metrics_by_config = {k: v for k, v in result['mean_metrics'].items() if k in filtered_configs}
    for run_config, metrics in mean_metrics_by_config.items():
        main_metrics = metrics.get('main', {})
        fractions = total_time_fractions.setdefault(run_config, {})
        total_time = main_metrics.get('totalTime') or 0
        if total_time == 0:
            continue
        for metric in ['transactionsSync', 'commitsSync']:
            fractions.setdefault(metric, []).append(main_metrics.get(metric, 0) / total_time)
    for [name_left, values_left], [name_right, values_right] \
            in combinations(mean_metrics_by_config.items(), 2):
        comparison_metrics = comparisons.setdefault(f'{name_left} vs {name_right}', {})
        for metric_group in ['main', 'committer', 'transactions_mean']:
            for metric, value_left in values_left.get(metric_group, {}).items():
                value_right = values_right.get(metric_group, {}).get(metric)
                if value_right and value_left:
                    (comparison_metrics.setdefault(metric_group, {})
                     .setdefault(metric, [])
                     .append(value_left / value_right))

for name, fractions in total_time_fractions.items():
    print(f'{name} total time fractions:\n')
    for k, v in fractions.items():
        print_summary_stats(k, v)
for name, metric_groups in comparisons.items():
    print(f"Comparing {name}:\n")
    for group_name, values in metric_groups.items():
        print(f'Metric group: {group_name}:\n')
        for metric, value in values.items():
            print_summary_stats(metric, value)

# archive_path = BASE_DIR.joinpath(f'{PROJECT_NAME}.zip')
# archive_path.touch(exist_ok=True)
# archive_file = ZipFile(archive_path, mode='w', compression=ZIP_DEFLATED, allowZip64=True)
# results_file = archive_file.open('results.json', mode='w', force_zip64=True)
# i = 0
# itr = results_db.itervalues()
# itr.seek_to_first()
# for v in itr:
#     results_file.write(v + b'\n')
#     print(f'zipping progress: {i / total}%')
#     # print(str(v, encoding='utf-8'))
#     i += 1
# results_file.close()
# archive_file.close()
