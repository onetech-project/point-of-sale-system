#!/usr/bin/env node

/**
 * Guest Checkout with QRIS Payment Flow
 * Simulates a guest customer ordering and paying via QRIS
 * 
 * Run: node guest-checkout-flow.js
 */

const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs');

const parsed = path.parse(__filename);
const resultPath = path.join(__dirname, '..', 'results', parsed.name);

if (!fs.existsSync(resultPath)){
  fs.mkdirSync(resultPath, { recursive: true });
}

// validate env variables
if (!process.env.MENU_PATH) {
  console.error('❌ ERROR: MENU_PATH environment variable is not set.');
  console.error('Please set MENU_PATH in your .env file to the guest menu path (e.g., /menu/:tenantId)');
  process.exit(1);
}

if (!process.env.CHECKOUT_PATH) {
  console.error('❌ ERROR: CHECKOUT_PATH environment variable is not set.');
  console.error('Please set CHECKOUT_PATH in your .env file to the guest checkout path (e.g., /checkout/:tenantId)');
  process.exit(1);
}

const config = {
  baseUrl: process.env.BASE_URL || 'http://localhost:3000',
  menuUrl: process.env.BASE_URL + process.env.MENU_PATH,
  checkoutUrl: process.env.BASE_URL + process.env.CHECKOUT_PATH,
  headless: false,
  slowMo: 10,
};

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

const ts = Date.now();

async function runGuestCheckoutFlow() {
  console.log('🛍️  Starting guest checkout automation...\n');
  
  const browser = await puppeteer.launch({
    headless: config.headless,
    slowMo: config.slowMo,
    defaultViewport: { width: 1280, height: 720 },
    args: ['--start-maximized']
  });

  const page = await browser.newPage();

  try {
    // ==================== STEP 1: OPEN GUEST MENU ====================
    console.log('📱 Step 1: Opening guest ordering page...');
    await page.goto(config.menuUrl, { waitUntil: 'networkidle2' });
    await sleep(1000);

    // ==================== STEP 2: BROWSE MENU ====================
    console.log('\n🍽️  Step 2: Browsing menu...');
    
    // Wait for menu items to load
    await page.waitForSelector('[data-testid="product-card"]', { timeout: 5000 })
      .catch(() => console.log('   ⚠️  Menu items not found with test IDs'));

    const menuItems = await page.$$('[data-testid="product-card"]');
    console.log(`   Found ${menuItems.length} menu items`);

    // ==================== STEP 3: ADD ITEMS TO CART ====================
    console.log('\n🛒 Step 3: Adding items to cart...');
    
    // Add max 3 different items
    for (let i = 0; i < Math.min(3, menuItems.length); i++) {
      const item = menuItems[i];
      
      // Get item name
      const itemName = await item.$eval(
        '[data-testid="product-name"]',
        el => el.textContent
      ).catch(() => `Item ${i + 1}`);
      
      console.log(`   Adding: ${itemName}`);
      
      // Click add to cart button
      const addBtn = await item.$('button[data-testid="add-to-cart-button"]');
      if (addBtn) {
        await addBtn.click();
        await sleep(800);
        console.log(`   ✅ Added ${itemName}`);
      }
    }

    // ==================== STEP 4: VIEW CART ====================
    console.log('\n🛒 Step 4: Viewing cart...');
    
    const cartTotal = await page.$eval(
      '[data-testid="cart-total"]',
      el => el.textContent
    ).catch(() => 'N/A');
    
    console.log(`   Cart Total: ${cartTotal}`);

    // ==================== STEP 5: PROCEED TO CHECKOUT ====================
    console.log('\n💳 Step 5: Proceeding to checkout...');
    
    const checkoutBtn = await page.$('button[data-testid="proceed-to-checkout-button"]');
    if (checkoutBtn) {
      await checkoutBtn.click();
      await sleep(1500);
    } else {
      console.log('   ⚠️  Checkout button not found, navigating manually...');
      await page.goto(config.checkoutUrl, { waitUntil: 'networkidle2' });
      await sleep(1500);
    }

    // ==================== STEP 6: FILL CUSTOMER INFO ====================
    console.log('\n👤 Step 6: Filling customer information...');
    
    const customerData = {
      name: 'Guest Customer ' + ts,
      phone: '081234567890',
      email: `guest${ts}@yopmail.com`,
      additionalNotes: 'Please deliver quickly'
    };

    const deliveryData = {
      street: 'Jl. Example No. 123',
      city: 'Jakarta',
      postalCode: '12345',
      province: 'DKI Jakarta',
      additionalNotes: 'Near the big mall'
    }

    const dineInData = {
      tableNumber: '5',
    }

    // choose delivery type
    await page.click('button[data-testid="delivery-type-pickup"]');
    await sleep(500);

    await page.type('input[data-testid="customer-name-input"]', customerData.name);
    await sleep(300);
    await page.type('input[data-testid="customer-phone-input"]', customerData.phone);
    await sleep(300);
    
    const emailInput = await page.$('input[data-testid="customer-email-input"]');
    if (emailInput) {
      await page.type('input[data-testid="customer-email-input"]', customerData.email);
      await sleep(300);
    }
    
    const tableInput = await page.$('input[data-testid="table-number-input"]');
    if (tableInput) {
      await page.type('input[data-testid="table-number-input"]', dineInData.tableNumber);
      await sleep(300);
    }

    const streetInput = await page.$('input[data-testid="address-street-input"]');
    if (streetInput) {
      await page.type('input[data-testid="address-street-input"]', deliveryData.street);
      await sleep(300);
      await page.type('input[data-testid="address-city-input"]', deliveryData.city);
      await sleep(300);
      await page.type('input[data-testid="address-province-input"]', deliveryData.province);
      await sleep(300);
      await page.type('input[data-testid="address-postalcode-input"]', deliveryData.postalCode);
      await sleep(300);
      await page.type('textarea[data-testid="address-notes-input"]', deliveryData.additionalNotes);
      await sleep(500);
    }

    const notesInput = await page.$('textarea[data-testid="additional-notes-input"]');
    if (notesInput) {
      await page.type('textarea[data-testid="additional-notes-input"]', customerData.additionalNotes);
      await sleep(500);
    }

    console.log('   ✅ Customer info filled:');
    console.log(`   Name: ${customerData.name}`);
    console.log(`   Phone: ${customerData.phone}`);
    console.log(`   Email: ${customerData.email}`);

    // ==================== STEP 7: ACCEPT TERMS ====================
    console.log('✅ Step 7: Accepting terms and conditions...');
    
    // only click if checkbox exists and not checked (multiple checkbox)
    const termsCheckboxes = await page.$$('input[type="checkbox"]');
    for (const checkbox of termsCheckboxes) {
      const isChecked = await page.evaluate(el => el.checked, checkbox);
      if (!isChecked) {
        await checkbox.click();
      }
    }
    await sleep(500);

    // ==================== STEP 8: PLACE ORDER ====================
    console.log('\n📤 Step 8: Placing order...');
    
    const placeOrderBtn = await page.$('button[data-testid="proceed-to-payment-button"]');
    if (placeOrderBtn) {
      await placeOrderBtn.click();
      await sleep(3000);

      // Wait for QRIS code or success page
      await page.waitForSelector(
        '[data-testid="payment-qris-qr-code"]',
        { timeout: 10000 }
      ).catch(() => console.log('   ⚠️  QRIS code not displayed'));

      console.log('   ✅ Order placed successfully!');
      await sleep(2000);

      // ==================== STEP 9: VERIFY QRIS DISPLAY ====================
      console.log('\n🔲 Step 9: Verifying QRIS display...');
      
      const qrisImg = await page.$('[data-testid="payment-qris-qr-code"]');
      if (qrisImg) {
        console.log('   ✅ QRIS code displayed');
      } else {
        console.log('   ⚠️  QRIS code not found');
      }

      // Get order details
      const orderReference = await page.$eval(
        '[data-testid="order-reference"]',
        el => el.textContent
      ).catch(() => 'N/A');
      
      console.log(`   📝 Order Reference: ${orderReference}`);
      // Take screenshot to ../results/guest-checkout-flow/result.png
      await page.screenshot({ path: path.join(resultPath, 'result.png'), fullPage: true });
      console.log('   📸 Screenshot saved to guest-checkout-qris.png');
    } else {
      console.log('   ⚠️  Place Order button not found');
    }

    // ==================== COMPLETE ====================
    console.log('\n🎉 ========================================');
    console.log('🎉  GUEST CHECKOUT COMPLETED!');
    console.log('🎉 ========================================');
    console.log('\n📋 Order Summary:');
    console.log(`   Customer: ${customerData.name}`);
    console.log(`   Phone: ${customerData.phone}`);
    console.log(`   Payment: QRIS`);
    console.log('========================================\n');

    await sleep(5000);

  } catch (error) {
    console.error('\n❌ Error during guest checkout:', error.message);
    await page.screenshot({ path: path.join(resultPath, 'guest-checkout-error.png'), fullPage: true });
    console.log('📸 Screenshot saved to guest-checkout-error.png');
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
runGuestCheckoutFlow().catch(console.error);
