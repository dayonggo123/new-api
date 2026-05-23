import re

with open('docs/电商图向导_种子数据_v3.sql', 'r', encoding='utf-8') as f:
    lines = f.readlines()

in_case_details = False
modified = False

for i, line in enumerate(lines):
    if 'INSERT INTO ecommerce_case_details' in line:
        in_case_details = True
        continue
    if not in_case_details:
        continue
    # Match lines that are the last field of a value tuple (case_reference)
    # Pattern: 3 spaces + quoted string ending with '),\n' or ');\n'
    if line.startswith("   '") and (line.rstrip().endswith("'),") or line.rstrip().endswith("');")):
        stripped = line.rstrip()
        if stripped.endswith("'),"):
            lines[i] = stripped[:-3] + "',\n   1779521421, 1779521421),\n"
        elif stripped.endswith("');"):
            lines[i] = stripped[:-3] + "',\n   1779521421, 1779521421);\n"
        modified = True

if modified:
    with open('docs/电商图向导_种子数据_v3.sql', 'w', encoding='utf-8') as f:
        f.writelines(lines)
    print('Fixed all case detail records')
else:
    print('No records needed fixing')
