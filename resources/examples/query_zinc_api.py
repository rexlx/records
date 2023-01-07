#!/usr/bin/env python3

"""
This serves as a very simple example of how to use the zinc api.

please refer to the docs for more advanced examples: https://docs.zincsearch.com
https://github.com/zinclabs/zinc
"""
import json
import requests as r
from datetime import datetime as dt
from requests.auth import HTTPBasicAuth

# i store my indexes by year month - name: 202212-IndexName
index = "ErcotSPP"
index = f"{dt.now().strftime('%Y%m')}-{index}"

# if you want to include date ranges, these give you the last month
#start = dt.today().replace(day=1)
#end = start + timedelta(days=30)

# use your url here
uri = f"""http://drfright.nullferatu.com:4080/api/{index}/_search"""

# secure authorization is outside the scope of this example. this is
# not suitable for production
a = HTTPBasicAuth("admin", "r0yalewithcheese")

q = {
        "search_type": "querystring",
        "query": 
        {
            "term": "LzHouston:>200",

        },
        "sort_fields": ["-@timestamp"],
        "from": 0,
        "max_results": 10,
        "aggs": {
            "max_bus": {
                "agg_type": "max",
                "field": "HbBusAvg"
                },
            "min_bus": {
                "agg_type": "min",
                "field": "HbBusAvg"
                },
            "avg_bus": {
                "agg_type": "avg",
                "field": "HbBusAvg"
                },
            "max_hub": {
                "agg_type": "max",
                "field": "HbHubAvg"
                },
            "min_hub": {
                "agg_type": "min",
                "field": "HbHubAvg"
                },
            "avg_hub": {
                "agg_type": "avg",
                "field": "HbHubAvg"
                }
        },
        "_source": []
    }

# post the query string to the uri using the auth object
res = r.post(uri, auth=a, data=json.dumps(q))
parsed_res = json.loads(res.text)
nicer_res = json.dumps(parsed_res, indent=4)
print(nicer_res)
