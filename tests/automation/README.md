# Real Browser Automation Scripts

These scripts automate real user interactions with your POS system - opening actual browsers, filling forms, clicking buttons, and completing full user journeys with real data.

## 🎯 What This Does

Unlike traditional test frameworks, these scripts:

- ✅ Open real Chrome browser (visible)
- ✅ Perform actual user actions (typing, clicking)
- ✅ Use real data (creates actual users, orders)
- ✅ Complete full workflows end-to-end
- ✅ Save screenshots on errors
- ✅ Log every step with details

**Think of it as a robot performing manual testing for you.**

## 📦 Installation

```bash
cd scripts/automation
npm install
```

This installs **Puppeteer** which includes Chromium browser.

## 🚀 Usage

### 1. User Registration Flow

Automates complete user registration with form filling:

```bash
npm run register
# or
node ./src/user-registration-flow.js
```

**What it does:**

1. Opens registration page
2. Fills business name, email, password, phone, address
3. Accepts terms & conditions
4. Submits form
5. Verifies success
6. Saves credentials to `test-users.json`

**Output:**

```
🚀 Starting automated user registration flow...
📍 Step 1: Opening registration page...
✍️  Step 2: Filling registration form...
   Business Name: Test Business 1737388800000
   Email: user1737388800000@test.com
   Phone: 081238880000
✅ Step 3: Accepting terms and conditions...
📤 Step 4: Submitting registration...
✅ ✅ ✅ REGISTRATION SUCCESSFUL! ✅ ✅ ✅
💾 Credentials saved to test-users.json
```

### 2. Guest Checkout Flow

Automates guest ordering with QRIS payment:

```bash
npm run guest
# or
node guest-checkout-flow.js
```

**What it does:**

1. Open guest menu page
2. Browse menu items
3. Add items to cart
4. View cart
5. Proceed to checkout
6. Fill customer info (name, phone, email)
7. Select QRIS payment
8. Place order
9. Verify QRIS display
10. Save QRIS screenshot

**Output:**

```
🛍️  Starting guest checkout automation...
📱 Step 1: Opening guest ordering page...
🍽️  Step 2: Browsing menu...
   Found 12 menu items
🛒 Step 3: Adding items to cart...
   Adding: Espresso
   ✅ Added Espresso
🔲 Step 9: Verifying QRIS display...
   ✅ QRIS code displayed
   📸 QRIS screenshot saved to qris-payment.png
🎉 GUEST CHECKOUT COMPLETED!
```

## ⚙️ Configuration

### Environment Variables

Create `.env` file:

```bash
# Application URL
BASE_URL=http://localhost:3000

# Test user credentials
TEST_EMAIL=admin@test.com
TEST_PASSWORD=admin123

# Browser settings
HEADLESS=false     # true = no browser window
SLOW_MO=100        # ms delay between actions
```

### Script Configuration

Edit the `config` object in each script:

```javascript
const config = {
  baseUrl: process.env.BASE_URL || 'http://localhost:3000',
  headless: false, // Set to true to hide browser
  slowMo: 150, // Slow down to see actions

  email: 'user@test.com',
  password: 'password123',
}
```

## 📸 Screenshots

Scripts automatically save screenshots on errors:

- `registration-error.png` - Registration failures
- `order-flow-error.png` - Order flow failures
- `guest-checkout-error.png` - Checkout failures
- `qris-payment.png` - QRIS code display

## 🔍 What Gets Created

### Real Data Created:

- ✅ **Users** - Actual user accounts in database
- ✅ **Products** - Real products with names, prices
- ✅ **Orders** - Complete orders with items
- ✅ **Guest Orders** - Guest checkout records
- ✅ **Payments** - QRIS payment intents

### Saved Files:

- `test-users.json` - All registered test users with credentials
- `*.png` - Screenshots of pages and errors

## 🎬 Visual Demonstration

Run with browser visible (default):

```bash
node complete-order-flow.js
```

You'll see:

- 🌐 Chrome browser opens
- 👁️ All actions performed visibly
- ⏱️ Slowed down to human speed
- 📝 Console logs each step
- ✅ Success indicators

## 🔧 Customization

### Add Custom Flows

Create `my-custom-flow.js`:

```javascript
const puppeteer = require('puppeteer')

async function runMyFlow() {
  const browser = await puppeteer.launch({
    headless: false,
    slowMo: 150,
  })

  const page = await browser.newPage()

  try {
    // Your automation steps here
    await page.goto('http://localhost:3000')
    await page.click('button#my-button')

    console.log('✅ Flow completed!')
  } catch (error) {
    console.error('❌ Error:', error)
  } finally {
    await browser.close()
  }
}

runMyFlow()
```

### Modify Existing Flows

Each script is self-contained and easy to modify:

1. Find the script file (e.g., `complete-order-flow.js`)
2. Modify the steps or selectors
3. Run it: `node complete-order-flow.js`

## 🐛 Troubleshooting

### Browser doesn't open

```bash
# Check if Puppeteer is installed
ls node_modules/puppeteer

# Reinstall
npm install puppeteer
```

### Element not found

```bash
# Update selectors in script
await page.$('button[data-testid="my-button"]');
# or
await page.$('button:has-text("Click Me")');
```

### Slow execution

```bash
# Reduce slowMo value
const config = {
  slowMo: 50,  // Faster (was 150)
};
```

### Script hangs

```bash
# Add timeout
await page.waitForSelector('button', { timeout: 5000 });
```

## 📊 Use Cases

### Development

- Test new features manually but faster
- Verify UI changes don't break flows
- Demo features to stakeholders

### QA Testing

- Regression testing after changes
- Smoke testing before deployment
- Load testing setup (run multiple instances)

### CI/CD Integration

```yaml
# .github/workflows/smoke-test.yml
- name: Run Smoke Tests
  run: |
    cd scripts/automation
    npm install
    npm run smoke-test
```

### Data Seeding

```bash
# Create 10 test orders
for i in {1..10}; do
  node complete-order-flow.js
done
```

## 🎯 Best Practices

1. **Run on clean data** - Use test database
2. **Check selectors** - Update if UI changes
3. **Save credentials** - Keep `test-users.json` safe
4. **Take screenshots** - Debug with visual evidence
5. **Use headless mode** - For CI/CD (`headless: true`)
6. **Add delays** - Don't go too fast (`slowMo`)

## 📚 Resources

- [Puppeteer Documentation](https://pptr.dev/)
- [Selector Reference](https://pptr.dev/#?product=Puppeteer&show=api-pageselector)
- [Examples](https://github.com/puppeteer/puppeteer/tree/main/examples)

## 🆘 Support

Issues? Check:

1. Is the app running? (`http://localhost:3000`)
2. Are selectors correct? (Check dev tools)
3. Is Puppeteer installed? (`npm install`)
4. Check error screenshots (`.png` files)

---

**Happy Automating! 🤖**
