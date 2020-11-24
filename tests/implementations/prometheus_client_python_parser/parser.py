import sys
from prometheus_client.openmetrics.parser import text_string_to_metric_families

input = sys.stdin.read()
families = text_string_to_metric_families(input)
print(list(families))
