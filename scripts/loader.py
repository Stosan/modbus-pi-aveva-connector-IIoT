import json
import yaml

# Load the PI Web API JSON items
with open('scripts/pitags.json', 'r') as f:
    pi_data = json.load(f)['Items']

# Create a lookup dictionary: { "Tagname": "WebId" }
webid_lookup = {item['Name']: item['WebId'] for item in pi_data}

# Load your YAML template
with open('config/config.yaml', 'r') as f:
    template = yaml.safe_load(f)

# Process the tags
for gateway in template['gateways']:
    if 'tags' in gateway:
        for tag in gateway['tags']:
            tag_name = tag['name']
            if tag_name in webid_lookup:
                tag['pi_web_id'] = webid_lookup[tag_name]
            else:
                tag['pi_web_id'] = "NOT_FOUND_IN_PI"

# Save the result
with open('final_integration_config.yaml', 'w') as f:
    yaml.dump(template, f, sort_keys=False)

print("Mapping complete!")