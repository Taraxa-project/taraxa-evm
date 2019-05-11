from pathlib import Path
import json
import rocksdb

json_file_path = Path('metrics.json')
json_file_path.touch(exist_ok=True)
json_file = json_file_path.open(mode='a')

db = rocksdb.DB('out/eth_metrics', rocksdb.Options(), read_only=True)

itr = db.itervalues()
itr.seek_to_first()
for v in itr:
    json_str = str(v, encoding='utf-8')
    json_file.write(json_str + '\n')
    print(json_str)
