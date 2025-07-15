
1. Create two event broker services with name `event-broker-service-1` and `event-broker-service-2`.
```bash
SC_USERNAME="${TEST_USERNAME}"
SC_PASSWORD=''

LOGIN_INFO="{\"username\": \"$SC_USERNAME\", \"password\": \"$SC_PASSWORD\"}"
#data-raw is important to ignore the @ simbol in the email
SC_TOKEN=$(curl "https://production-api.solace.cloud/api/v0/iam/tokens" \
    -X POST \
    -H 'Content-Type: application/json' \
    --data-raw "${LOGIN_INFO}" | jq .token -r)
    
PAYLOAD=$( jq -n --arg name "event-broker-service-1" '{
	"name":$name,
    "serviceClassId":"ENTERPRISE_250_STANDALONE",
    "eventBrokerVersion": "10.8",
    "datacenterId":"gke-gcp-us-central1-a",
}')
curl --location 'https://production-api.solace.cloud/api/v2/missionControl/eventBrokerServices' \
--header "Authorization: Bearer $SC_TOKEN" \
--header 'Content-Type: application/json' \
--data $PAYLOAD &

PAYLOAD=$( jq -n --arg name "event-broker-service-2" '{
	"name":$name,
    "serviceClassId":"ENTERPRISE_250_STANDALONE",
    "eventBrokerVersion": "10.8",
    "datacenterId":"gke-gcp-us-central1-a",
}')
curl --location 'https://production-api.solace.cloud/api/v2/missionControl/eventBrokerServices' \
--header "Authorization: Bearer $SC_TOKEN" \
--header 'Content-Type: application/json' \
--data $PAYLOAD &


curl --location 'https://production-api.solace.cloud/api/v2/missionControl/eventBrokerServices' \
--header "Authorization: Bearer $SC_TOKEN" \
--header 'Content-Type: application/json' 
`
```

2. wait for services to create

3. update import.tf with correct service ids

4. setup terraform environment variables
```bash
SC_USERNAME="${TEST_USERNAME}"
SC_PASSWORD=''

LOGIN_INFO="{\"username\": \"$SC_USERNAME\", \"password\": \"$SC_PASSWORD\"}"
#data-raw is important to ignore the @ simbol in the email
export SOLACECLOUD_API_TOKEN=$(curl "https://production-api.solace.cloud/api/v0/iam/tokens" \
    -X POST \
    -H 'Content-Type: application/json' \
    --data-raw "${LOGIN_INFO}" | jq .token -r)

export SOLACE_BASE_URL="https://production-api.solace.cloud"

curl --location "$SOLACE_BASE_URL/api/v2/missionControl/eventBrokerServices" \
--header "Authorization: Bearer $SOLACECLOUD_API_TOKEN" \
--header 'Content-Type: application/json' 
```

5. terraform plan. Output should look like example.txt. Notice that all the state is defined as a terraform file. It's necessary for a user to remove the excess variables

6. terraform plan -generate-config-out="generated_resources.tf" does work

