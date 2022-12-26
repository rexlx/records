#!/usr/bin/env python3

"""
This servers as very simple example of how to use the zinc api.

please refer to the docs for more advanced examples: https://docs.zincsearch.com
https://github.com/zinclabs/zinc
"""
import json
import requests as r
from datetime import datetime as dt
from requests.auth import HTTPBasicAuth
from datetime import timedelta

index = "ErcotSPP"
# i store my indexes by year month - name: 202212-IndexName
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
            "term": "LzHouston:>100",

        },
        "sort_fields": ["-@timestamp"],
        "from": 0,
        "max_results": 100,
        "aggs": {
            "max_SPP": {
                "agg_type": "max",
                "field": "LzHouston"
                },
            "min_SPP": {
                "agg_type": "min",
                "field": "LzHouston"
                },
            "avg_SPP": {
                "agg_type": "avg",
                "field": "LzHouston"
                }
        },
        "_source": []
    }

# post the query string to the uri using the auth object
res = r.post(uri, auth=a, data=json.dumps(q))

print(res.text)
