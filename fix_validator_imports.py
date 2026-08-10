import re

with open('internal/utils/helper.go', 'r') as f:
    content = f.read()

# Make sure the import is there. Let's just find the imports block and ensure go-playground/validator is in it.
if '"github.com/go-playground/validator/v10"' not in content:
    content = re.sub(
        r'import \(\n',
        'import (\n\t"github.com/go-playground/validator/v10"\n',
        content,
        count=1
    )

with open('internal/utils/helper.go', 'w') as f:
    f.write(content)
