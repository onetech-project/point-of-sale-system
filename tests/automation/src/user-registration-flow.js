#!/usr/bin/env node

/**
 * Automated User Registration Flow
 * This script opens a real browser and performs complete user registration
 * Run: node user-registration-flow.js
 */

const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs');

const parsed = path.parse(__filename);
const resultPath = path.join(__dirname, '..', 'results', parsed.name);

if (!fs.existsSync(resultPath)){
  fs.mkdirSync(resultPath, { recursive: true });
}

const config = {
  baseUrl: process.env.BASE_URL || 'http://localhost:3000',
  headless: false, // Show browser
  slowMo: 10, // Slow down by 10ms to see actions
};

// Generate random user data
function generateUserData() {
  const timestamp = Date.now();
  return {
    businessName: `Test Business ${timestamp}`,
    email: `owner.business${timestamp}@yopmail.com`,
    firstName: 'Owner',
    lastName: `Business ${timestamp}`,
    password: 'P@ssw0rd',
    ts: timestamp
  };
}

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function runRegistrationFlow() {
  console.log('🚀 Starting automated user registration flow...\n');
  
  const browser = await puppeteer.launch({
    headless: config.headless,
    slowMo: config.slowMo,
    defaultViewport: { width: 1280, height: 720 },
    args: ['--start-maximized']
  });

  const page = await browser.newPage();
  const userData = generateUserData();

  try {
    // Step 1: Navigate to registration page
    console.log('📍 Step 1: Opening registration page...');
    await page.goto(`${config.baseUrl}/signup`, { waitUntil: 'networkidle2' });
    await sleep(1000);

    // Step 2: Fill registration form
    console.log('✍️  Step 2: Filling registration form...');
    console.log(`   Business Name: ${userData.businessName}`);
    console.log(`   Email: ${userData.email}`);
    console.log(`   Phone: ${userData.phone}`);
    console.log(`   Password: ${userData.password}`);
    console.log(`   First Name: ${userData.firstName}`);
    console.log(`   Last Name: ${userData.lastName}`);

    await page.type('input[name="businessName"]', userData.businessName);
    await sleep(500);
    
    await page.type('input[name="email"]', userData.email);
    await sleep(500);
    
    await page.type('input[name="firstName"]', userData.firstName);
    await sleep(500);

    await page.type('input[name="lastName"]', userData.lastName);
    await sleep(500);

    await page.type('input[name="password"]', userData.password);
    await sleep(500);
    
    await page.type('input[name="confirmPassword"]', userData.password);
    await sleep(500);
    

    // Step 3: Accept terms
    console.log('✅ Step 3: Accepting terms and conditions...');
    // only click if checkbox exists and not checked (multiple checkbox)
    const termsCheckboxes = await page.$$('input[type="checkbox"]');
    for (const checkbox of termsCheckboxes) {
      const isChecked = await page.evaluate(el => el.checked, checkbox);
      if (!isChecked) {
        await checkbox.click();
      }
    }
    await sleep(500);

    // Step 4: Submit form
    console.log('📤 Step 4: Submitting registration...');
    await page.click('button[type="submit"]');
    
    // Wait for response
    await page.waitForNavigation({ waitUntil: 'networkidle2', timeout: 10000 });
    await sleep(2000);

    // Step 5: Check if registration successful
    const currentUrl = page.url();
    console.log(`📍 Current URL: ${currentUrl}`);

    if (currentUrl.includes('/login')) {
      console.log('\n✅ ✅ ✅ REGISTRATION SUCCESSFUL! ✅ ✅ ✅\n');
      console.log('📋 User Details:');
      console.log(`   Business: ${userData.businessName}`);
      console.log(`   Email: ${userData.email}`);
      console.log(`   Password: ${userData.password}`);
      console.log(`   Phone: ${userData.phone}`);
      console.log(`   First Name: ${userData.firstName}`);
      console.log(`   Last Name: ${userData.lastName}`);
      
      // Save credentials to file
      const fs = require('fs');
      const credentials = {
        timestamp: new Date().toISOString(),
        ...userData
      };
      const jsonPath = path.join(resultPath, 'results.json');
      fs.appendFileSync(
        jsonPath,
        JSON.stringify(credentials, null, 2) + ',\n'
      );
      console.log(`\n💾 Credentials saved to ${jsonPath}`);
    } else {
      console.log('\n⚠️  Registration may have failed or needs email verification');
    }

    await sleep(2000);

  } catch (error) {
    console.error('\n❌ Error during registration:', error.message);
    const errorPath = path.join(resultPath, `error-${userData.ts}.png`);
    await page.screenshot({ path: errorPath, fullPage: true });
    console.log(`📸 Screenshot saved to ${errorPath}`);
  } finally {
    if (!config.headless) {
      console.log('\n⏳ Browser will close in 1 seconds...');
      await sleep(1000);
    }
    await browser.close();
    console.log('✅ Browser closed');
  }
}

// Run the automation
runRegistrationFlow().catch(console.error);
