const fs = require('fs');
const path = require('path');

const API_URL = 'http://localhost:8080/api/proxies';
const LOCAL_FILE = path.join(__dirname, 'local.txt');

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function addProxy(apiKey) {
  const proxyData = {
    api_key: apiKey,
    service_type: 'kiotproxy',
    min_time_reset: 370
  };

  let attempt = 1;

  while (true) {
    console.log(`  [Attempt ${attempt}] Adding proxy for key: ${apiKey}`);

    try {
      const response = await fetch(API_URL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(proxyData)
      });

      const result = await response.json();

      if (result.id) {
        console.log(`  ✅ Success! ID: ${result.id}, Proxy: ${result.proxy_str}\n`);
        return true;
      } else {
        console.log(`  ❌ Failed:`, JSON.stringify(result));
      }
    } catch (error) {
      console.log(`  ❌ Error: ${error.message}`);
    }

    attempt++;
    console.log(`  Waiting 2s before retry...`);
    await sleep(2000);
  }
}

async function main() {
  // Đọc file local.txt
  const content = fs.readFileSync(LOCAL_FILE, 'utf-8');
  const apiKeys = content.split('\n').map(line => line.trim()).filter(line => line.length > 0);

  console.log(`Found ${apiKeys.length} API keys in local.txt\n`);

  for (let i = 0; i < apiKeys.length; i++) {
    const apiKey = apiKeys[i];
    console.log(`[${i + 1}/${apiKeys.length}] Processing: ${apiKey}`);
    await addProxy(apiKey);
  }

  console.log('🎉 All proxies added successfully!');
}

main();
