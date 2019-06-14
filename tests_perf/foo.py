import json
from zipfile import ZipFile

block_numbers = set()
for line in ZipFile('metrics.zip').open('results.json'):
    result = json.loads(line)
    block_num = result['blockNumber']
    assert block_num not in block_numbers
    print(block_num)
    block_numbers.add(block_num)
