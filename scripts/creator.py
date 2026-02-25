import yaml
import csv

# Path to your yaml file
INPUT_FILE = '../config/config.yaml'
OUTPUT_FILE = 'pi_points_import.csv'

def convert_yaml_to_pi_csv(input_path, output_path):
    with open(input_path, 'r') as f:
        # Load the YAML data
        data = yaml.safe_load(f)

    # Standard PI Builder Headers
    # Adjust 'PointSource' and 'Location' codes based on your specific interface requirements
    headers = [
        "tag", "pointtype", "pointsource", "instrumenttag", 
        "engunits", "descriptor", "location1", "location2", "location3", "location4", "location5", "selected"
    ]

    pi_points = []
 
    # Iterate through the gateways (Gateway_1, Gateway_2, etc.)
    for gateway in data['gateways']:
        gateway_name = gateway.get('name', 'Unknown_Gateway')
        tags = gateway.get('tags', [])

        for tag in tags:
            # Map YAML fields to PI Attributes
            pi_points.append({
                "tag": tag.get('name'),
                "pointtype": "Float32",
                "pointsource": "PIWebAPI_OMF",          # Typically 'M' for Modbus or 'L' for Lab
                "instrumenttag": tag.get('register'),
                "engunits": tag.get('device_type'),
                "descriptor": f"Register {tag.get('register')} from {gateway_name}",
                "location1": 4,              # Interface ID (Update as needed)
                "location2": 0,              # Scan Class (Update as needed)
                "location3": 1,              # Scan Class (Update as needed)
                "location4": 1,              # Scan Class (Update as needed)
                "location5": 0,              # Scan Class (Update as needed)
                "selected": "X"
            })

    # Write to CSV
    with open(output_path, 'w', newline='') as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=headers)
        writer.writeheader()
        writer.writerows(pi_points)

    print(f"Success! {len(pi_points)} points exported to {output_path}")

if __name__ == "__main__":
    convert_yaml_to_pi_csv(INPUT_FILE, OUTPUT_FILE)