# Login Verification Redirect - UX Improvement

**Date**: 2025-11-23  
**Feature**: Redirect unverified users to verification page on login attempt  
**Status**: ✅ Specified and Integrated  
**Related**: EMAIL_VERIFICATION_FEATURE.md

## Summary

Enhanced the login flow to gracefully handle unverified users. Instead of blocking with an error, we now redirect them to a dedicated "Verification Required" page where they can easily resend the verification email.

---

## 🎯 Problem Statement

**Before**:
- User registers but doesn't verify email
- User tries to login later
- Gets blocked with error message
- No clear path to resend verification
- Poor user experience

**After**:
- User registers but doesn't verify email
- User tries to login
- Gets redirected to verification-required page
- Clear message with one-click resend
- Email pre-filled from login attempt
- Smooth UX with clear next steps

---

## 🔄 Updated User Flow

### Scenario: Unverified User Tries to Login

```
┌─────────────────────────────────────────┐
│ 1. User enters email + password         │
│    on /auth/login page                  │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ 2. Backend validates credentials        │
│    ✅ Password correct                   │
│    ❌ Email not verified                 │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ 3. Backend returns 403 response:        │
│    {                                    │
│      "error": "EMAIL_NOT_VERIFIED",     │
│      "message": "Please verify email",  │
│      "email": "user@example.com",       │
│      "can_resend": true                 │
│    }                                    │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ 4. Frontend catches error and redirects │
│    to /auth/verification-required       │
│    with email in query params           │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ 5. Verification Required Page shows:    │
│    - Clear message                      │
│    - User's email (pre-filled)          │
│    - "Resend Verification" button       │
│    - "Check Spam" instructions          │
│    - Link to support                    │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ 6. User clicks "Resend Verification"    │
│    - New email sent                     │
│    - Success message shown              │
│    - Instructions to check inbox        │
└─────────────────────────────────────────┘
```

---

## 🔧 Implementation Details

### Backend Changes (T361/T341)

**File**: `backend/auth-service/src/api/handlers.go`

**Update Login Handler**:

```go
// POST /api/auth/login handler
func (h *AuthHandler) Login(c echo.Context) error {
    var req LoginRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, map[string]string{"error": "Invalid request"})
    }
    
    // Validate credentials
    user, err := h.authService.ValidateCredentials(c.Request().Context(), req.Email, req.Password, req.TenantID)
    if err != nil {
        return c.JSON(401, map[string]string{"error": "Invalid credentials"})
    }
    
    // ✨ NEW: Check email verification status
    if !user.EmailVerified {
        // Check feature flag
        requireVerification := os.Getenv("REQUIRE_EMAIL_VERIFICATION") == "true"
        
        if requireVerification {
            // BLOCK LOGIN - Hard enforcement
            return c.JSON(403, map[string]interface{}{
                "error": "EMAIL_NOT_VERIFIED",
                "code": "EMAIL_NOT_VERIFIED",
                "message": "Please verify your email address to continue",
                "email": user.Email,
                "can_resend": true,
                "verification_required": true,
            })
        } else {
            // ALLOW LOGIN - Soft encouragement (log warning)
            h.logger.Warn("User logged in without email verification",
                "user_id", user.ID,
                "email", user.Email,
                "tenant_id", user.TenantID,
            )
        }
    }
    
    // Continue with normal login flow
    session, err := h.sessionService.Create(c.Request().Context(), user.ID, user.TenantID)
    if err != nil {
        return c.JSON(500, map[string]string{"error": "Failed to create session"})
    }
    
    // Publish login event
    h.eventPublisher.PublishUserLogin(
        c.Request().Context(),
        user.TenantID,
        user.ID,
        user.Email,
        user.Name,
        c.RealIP(),
        c.Request().UserAgent(),
    )
    
    // Set session cookie
    h.setSessionCookie(c, session.Token)
    
    return c.JSON(200, map[string]interface{}{
        "message": "Login successful",
        "user": user,
        "email_verified": user.EmailVerified, // Include verification status
    })
}
```

**Response Formats**:

```json
// Success (verified user)
{
  "message": "Login successful",
  "user": { ... },
  "email_verified": true
}

// Success (unverified, soft enforcement)
{
  "message": "Login successful",
  "user": { ... },
  "email_verified": false
}

// Error (unverified, hard enforcement)
{
  "error": "EMAIL_NOT_VERIFIED",
  "code": "EMAIL_NOT_VERIFIED",
  "message": "Please verify your email address to continue",
  "email": "user@example.com",
  "can_resend": true,
  "verification_required": true
}
```

---

### Frontend Changes (T362, T363, T364, T365)

#### 1. Update Login Page (T362/T365)

**File**: `frontend/src/pages/auth/login.tsx`

```tsx
import { useState } from 'react';
import { useRouter } from 'next/router';
import { useTranslation } from 'react-i18next';
import { authService } from '@/services/auth.service';

export default function Login() {
  const { t } = useTranslation();
  const router = useRouter();
  const [formData, setFormData] = useState({ email: '', password: '', tenantId: '' });
  const [errors, setErrors] = useState({});
  const [isLoading, setIsLoading] = useState(false);
  const [serverError, setServerError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setServerError('');
    setIsLoading(true);

    try {
      const response = await authService.login({
        email: formData.email,
        password: formData.password,
        tenantId: formData.tenantId
      });
      
      // Check if email is not verified (soft enforcement)
      if (response.email_verified === false) {
        // Show warning banner but allow access
        sessionStorage.setItem('show_verification_banner', 'true');
      }

      // Login successful - redirect to dashboard
      router.push('/dashboard');
      
    } catch (error: any) {
      console.error('Login error:', error);
      
      // ✨ NEW: Handle email not verified error
      if (error.response?.data?.code === 'EMAIL_NOT_VERIFIED') {
        // Redirect to verification required page with email
        const email = error.response.data.email || formData.email;
        router.push({
          pathname: '/auth/verification-required',
          query: { email }
        });
        return;
      }
      
      // Handle other errors
      setServerError(
        error.response?.data?.message || 
        t('auth.login.errors.invalidCredentials')
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            {t('auth.login.title')}
          </h2>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          {serverError && (
            <div className="rounded-md bg-red-50 p-4">
              <div className="text-sm text-red-800">
                {serverError}
              </div>
            </div>
          )}

          {/* Form fields */}
          {/* ... existing input fields ... */}

          <div>
            <button
              type="submit"
              disabled={isLoading}
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
            >
              {isLoading ? t('common.loading') : t('auth.login.submit')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

#### 2. Create Verification Required Page (T363/T364)

**File**: `frontend/src/pages/auth/verification-required.tsx`

```tsx
import { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import { useTranslation } from 'react-i18next';
import { authService } from '@/services/auth.service';
import Button from '@/components/ui/Button';

export default function VerificationRequired() {
  const { t } = useTranslation();
  const router = useRouter();
  const { email } = router.query;
  
  const [isResending, setIsResending] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [countdown, setCountdown] = useState(0);

  useEffect(() => {
    if (!email) {
      // No email provided, redirect to login
      router.push('/login');
    }
  }, [email, router]);

  useEffect(() => {
    if (countdown > 0) {
      const timer = setTimeout(() => setCountdown(countdown - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [countdown]);

  const handleResend = async () => {
    if (countdown > 0) return;
    
    setIsResending(true);
    setMessage(null);

    try {
      await authService.resendVerification(email as string);
      setMessage({
        type: 'success',
        text: t('auth.verification.resendSuccess')
      });
      setCountdown(60); // 60 second cooldown
    } catch (error: any) {
      setMessage({
        type: 'error',
        text: error.response?.data?.error || t('auth.verification.resendError')
      });
    } finally {
      setIsResending(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        {/* Header */}
        <div className="text-center">
          <div className="mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-yellow-100">
            <svg className="h-8 w-8 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          </div>
          
          <h2 className="mt-6 text-3xl font-extrabold text-gray-900">
            {t('auth.verification.requiredTitle')}
          </h2>
          
          <p className="mt-2 text-sm text-gray-600">
            {t('auth.verification.requiredDescription')}
          </p>
        </div>

        {/* Email display */}
        <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
          <p className="text-sm font-medium text-blue-900">
            {t('auth.verification.emailSentTo')}
          </p>
          <p className="mt-1 text-base font-semibold text-blue-700">
            {email}
          </p>
        </div>

        {/* Message display */}
        {message && (
          <div className={`rounded-md p-4 ${
            message.type === 'success' 
              ? 'bg-green-50 border border-green-200' 
              : 'bg-red-50 border border-red-200'
          }`}>
            <p className={`text-sm ${
              message.type === 'success' ? 'text-green-800' : 'text-red-800'
            }`}>
              {message.text}
            </p>
          </div>
        )}

        {/* Instructions */}
        <div className="bg-white shadow-sm border border-gray-200 rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">
            {t('auth.verification.nextSteps')}
          </h3>
          
          <ol className="space-y-3 text-sm text-gray-700">
            <li className="flex items-start">
              <span className="flex-shrink-0 h-6 w-6 flex items-center justify-center rounded-full bg-indigo-100 text-indigo-600 font-semibold text-xs mr-3">
                1
              </span>
              <span>{t('auth.verification.step1')}</span>
            </li>
            <li className="flex items-start">
              <span className="flex-shrink-0 h-6 w-6 flex items-center justify-center rounded-full bg-indigo-100 text-indigo-600 font-semibold text-xs mr-3">
                2
              </span>
              <span>{t('auth.verification.step2')}</span>
            </li>
            <li className="flex items-start">
              <span className="flex-shrink-0 h-6 w-6 flex items-center justify-center rounded-full bg-indigo-100 text-indigo-600 font-semibold text-xs mr-3">
                3
              </span>
              <span>{t('auth.verification.step3')}</span>
            </li>
          </ol>
        </div>

        {/* Actions */}
        <div className="space-y-4">
          <Button
            onClick={handleResend}
            disabled={isResending || countdown > 0}
            fullWidth
            variant="primary"
          >
            {countdown > 0 
              ? t('auth.verification.resendCountdown', { seconds: countdown })
              : isResending 
                ? t('common.loading')
                : t('auth.verification.resendButton')
            }
          </Button>

          <div className="text-center">
            <button
              onClick={() => router.push('/login')}
              className="text-sm font-medium text-indigo-600 hover:text-indigo-500"
            >
              {t('auth.verification.backToLogin')}
            </button>
          </div>
        </div>

        {/* Help section */}
        <div className="bg-gray-50 border border-gray-200 rounded-md p-4">
          <h4 className="text-sm font-medium text-gray-900 mb-2">
            {t('auth.verification.didntReceive')}
          </h4>
          <ul className="text-sm text-gray-600 space-y-1 list-disc list-inside">
            <li>{t('auth.verification.checkSpam')}</li>
            <li>{t('auth.verification.checkTypo')}</li>
            <li>{t('auth.verification.waitFewMinutes')}</li>
          </ul>
          
          <p className="mt-3 text-sm text-gray-600">
            {t('auth.verification.stillIssues')}{' '}
            <a href="mailto:support@pos-system.com" className="font-medium text-indigo-600 hover:text-indigo-500">
              {t('auth.verification.contactSupport')}
            </a>
          </p>
        </div>
      </div>
    </div>
  );
}
```

---

### Translations (Update T351-T352)

#### English (`frontend/public/locales/en/auth.json`)

```json
{
  "verification": {
    "requiredTitle": "Verify Your Email",
    "requiredDescription": "We sent a verification email to activate your account",
    "emailSentTo": "Verification email sent to:",
    "nextSteps": "What to do next:",
    "step1": "Check your email inbox (and spam folder)",
    "step2": "Click the verification link in the email",
    "step3": "Return here to login",
    "resendCountdown": "Resend in {{seconds}}s",
    "backToLogin": "← Back to Login",
    "didntReceive": "Didn't receive the email?",
    "checkSpam": "Check your spam or junk folder",
    "checkTypo": "Make sure your email address is correct",
    "waitFewMinutes": "Wait a few minutes and try resending",
    "stillIssues": "Still having issues?",
    "contactSupport": "Contact Support"
  }
}
```

#### Indonesian (`frontend/public/locales/id/auth.json`)

```json
{
  "verification": {
    "requiredTitle": "Verifikasi Email Anda",
    "requiredDescription": "Kami telah mengirim email verifikasi untuk mengaktifkan akun Anda",
    "emailSentTo": "Email verifikasi dikirim ke:",
    "nextSteps": "Langkah selanjutnya:",
    "step1": "Periksa kotak masuk email Anda (dan folder spam)",
    "step2": "Klik tautan verifikasi di email",
    "step3": "Kembali ke sini untuk login",
    "resendCountdown": "Kirim ulang dalam {{seconds}}d",
    "backToLogin": "← Kembali ke Login",
    "didntReceive": "Tidak menerima email?",
    "checkSpam": "Periksa folder spam atau junk",
    "checkTypo": "Pastikan alamat email Anda benar",
    "waitFewMinutes": "Tunggu beberapa menit dan coba kirim ulang",
    "stillIssues": "Masih mengalami masalah?",
    "contactSupport": "Hubungi Dukungan"
  }
}
```

---

## 🎨 UI/UX Design

### Verification Required Page Components

```
┌─────────────────────────────────────────────────┐
│                  [Email Icon]                    │
│                                                  │
│            Verify Your Email                     │
│   We sent a verification email to activate      │
│              your account                        │
│                                                  │
├─────────────────────────────────────────────────┤
│  📧 Verification email sent to:                 │
│     user@example.com                            │
├─────────────────────────────────────────────────┤
│                                                  │
│  What to do next:                               │
│  1️⃣ Check your email inbox (and spam)          │
│  2️⃣ Click the verification link                │
│  3️⃣ Return here to login                       │
│                                                  │
├─────────────────────────────────────────────────┤
│                                                  │
│  [ Resend Verification Email ]  (Primary Button)│
│                                                  │
│          ← Back to Login (Link)                 │
│                                                  │
├─────────────────────────────────────────────────┤
│  Didn't receive the email?                      │
│  • Check your spam or junk folder               │
│  • Make sure email address is correct           │
│  • Wait a few minutes and try resending         │
│                                                  │
│  Still having issues? Contact Support           │
└─────────────────────────────────────────────────┘
```

### Features:
1. **Clear messaging**: User knows exactly what happened
2. **Email display**: Shows which email needs verification
3. **Step-by-step guide**: Clear instructions
4. **One-click resend**: Easy to request new email
5. **Cooldown timer**: Prevents spam (60 seconds)
6. **Help section**: Troubleshooting tips
7. **Support link**: Easy access to help
8. **Back to login**: Easy navigation

---

## 🧪 Testing

### Test Scenarios

#### T366: Unit Test - Login Handler Email Verification Check

```go
// backend/tests/unit/api/login_handler_test.go

func TestLogin_UnverifiedEmail_HardEnforcement(t *testing.T) {
    os.Setenv("REQUIRE_EMAIL_VERIFICATION", "true")
    
    // Setup
    handler := setupAuthHandler()
    user := createTestUser(emailVerified: false)
    
    // Execute
    req := httptest.NewRequest("POST", "/api/auth/login", createLoginBody(user))
    rec := httptest.NewRecorder()
    handler.Login(echo.New().NewContext(req, rec))
    
    // Assert
    assert.Equal(t, 403, rec.Code)
    
    var response map[string]interface{}
    json.Unmarshal(rec.Body.Bytes(), &response)
    
    assert.Equal(t, "EMAIL_NOT_VERIFIED", response["code"])
    assert.Equal(t, user.Email, response["email"])
    assert.True(t, response["can_resend"].(bool))
}

func TestLogin_UnverifiedEmail_SoftEnforcement(t *testing.T) {
    os.Setenv("REQUIRE_EMAIL_VERIFICATION", "false")
    
    // Setup
    handler := setupAuthHandler()
    user := createTestUser(emailVerified: false)
    
    // Execute
    req := httptest.NewRequest("POST", "/api/auth/login", createLoginBody(user))
    rec := httptest.NewRecorder()
    handler.Login(echo.New().NewContext(req, rec))
    
    // Assert
    assert.Equal(t, 200, rec.Code) // Login succeeds
    
    var response map[string]interface{}
    json.Unmarshal(rec.Body.Bytes(), &response)
    
    assert.False(t, response["email_verified"].(bool)) // But flag is set
}
```

#### T367: E2E Test - Unverified User Login Flow

```typescript
// frontend/tests/e2e/unverified-login.spec.ts

import { test, expect } from '@playwright/test';

test('unverified user redirected to verification page on login', async ({ page }) => {
  // Navigate to login
  await page.goto('/login');
  
  // Enter credentials for unverified user
  await page.fill('input[name="email"]', 'unverified@test.com');
  await page.fill('input[name="password"]', 'password123');
  await page.click('button[type="submit"]');
  
  // Should redirect to verification required page
  await expect(page).toHaveURL(/\/auth\/verification-required/);
  
  // Should show user's email
  await expect(page.locator('text=unverified@test.com')).toBeVisible();
  
  // Should show resend button
  const resendButton = page.locator('button:has-text("Resend Verification Email")');
  await expect(resendButton).toBeVisible();
  
  // Click resend
  await resendButton.click();
  
  // Should show success message
  await expect(page.locator('text=Verification email sent')).toBeVisible();
  
  // Should show countdown
  await expect(page.locator('text=Resend in')).toBeVisible();
});

test('back to login link works', async ({ page }) => {
  await page.goto('/auth/verification-required?email=test@test.com');
  
  await page.click('text=Back to Login');
  
  await expect(page).toHaveURL('/login');
});
```

---

## 📊 Tasks Summary

### New Tasks Added: 5 tasks

| Task | Phase | Description | Time |
|------|-------|-------------|------|
| T361 | 4 (US2) | Add email verification check to login handler (backend) | 1h |
| T362 | 4 (US2) | Add redirect logic to login page (frontend) | 30min |
| T363 | 4 (US2) | Create verification-required page (frontend) | 2h |
| T364 | 15 | Create verification-required page in Phase 15 (duplicate of T363) | - |
| T365 | 15 | Update login page error handling in Phase 15 (duplicate of T362) | - |
| T366 | 15 | Unit tests for login verification check | 1h |
| T367 | 15 | E2E tests for unverified login flow | 1h |

**Note**: T364 and T365 are duplicates consolidated into T361-T363. Total unique tasks: 5

### Updated Totals

- **Previous Total**: 363 tasks
- **New Tasks Added**: +5 tasks (T361-T363, T366-T367)
- **New Total**: **368 tasks**

---

## 🔄 Integration Points

### With Existing Features

1. **Email Verification (Phase 15)**:
   - Uses T339 (verify-email endpoint)
   - Uses T340 (resend-verification endpoint)
   - Uses T350 (auth service methods)
   - Uses T351-T352 (translations)

2. **Notification Service**:
   - Uses T302 (registration email template)
   - Uses T310 (user.registered event with token)

3. **Login Flow (Phase 4)**:
   - Enhances T090 (login handler)
   - Enhances T095 (login page)
   - Uses T096 (auth service)

---

## ⚙️ Configuration

### Feature Flag Behavior

```bash
# Soft Enforcement (Default - Recommended for MVP)
REQUIRE_EMAIL_VERIFICATION=false

# Behavior:
# - User can login without verification
# - Warning logged in backend
# - Banner shown in dashboard
# - email_verified field returned in response
```

```bash
# Hard Enforcement (Recommended for Production)
REQUIRE_EMAIL_VERIFICATION=true

# Behavior:
# - User CANNOT login without verification
# - 403 error returned
# - Redirected to verification-required page
# - Must verify to access system
```

---

## 🎯 Benefits

### User Experience
- ✅ Clear feedback on why login failed
- ✅ Guided path to resolution (verification)
- ✅ One-click resend (no need to navigate)
- ✅ Email pre-filled from login attempt
- ✅ Helpful troubleshooting tips
- ✅ Easy way back to login

### Technical
- ✅ Proper error handling
- ✅ Graceful UX degradation
- ✅ Feature flag for gradual rollout
- ✅ Prevents confusion
- ✅ Reduces support tickets
- ✅ Maintains security

### Business
- ✅ Higher conversion rate (less drop-off)
- ✅ Better onboarding experience
- ✅ Fewer abandoned accounts
- ✅ Clearer user expectations
- ✅ Professional appearance

---

## 📋 Implementation Checklist

### Backend (T361, T366)
- [ ] Update login handler with verification check
- [ ] Return proper error response with email and metadata
- [ ] Respect feature flag (hard vs soft enforcement)
- [ ] Log unverified login attempts
- [ ] Unit tests for verification check
- [ ] Integration tests for error responses

### Frontend (T362, T363, T367)
- [ ] Update login page error handling
- [ ] Implement redirect to verification-required
- [ ] Create verification-required page
- [ ] Add translations (EN/ID)
- [ ] Implement resend with cooldown
- [ ] Add help section
- [ ] Unit tests for components
- [ ] E2E tests for full flow

### Testing
- [ ] Test with feature flag = false (soft)
- [ ] Test with feature flag = true (hard)
- [ ] Test resend functionality
- [ ] Test cooldown timer
- [ ] Test navigation (back to login)
- [ ] Test email display
- [ ] Test error scenarios

---

## 🚀 Deployment Strategy

### Phase 1: MVP Launch
```bash
# Soft enforcement - don't block users
REQUIRE_EMAIL_VERIFICATION=false
```
- Users can login without verification
- Collect metrics on verification rates
- Show banner to encourage verification

### Phase 2: Gradual Enforcement (Week 2-4)
```bash
# Still soft, but prepare infrastructure
REQUIRE_EMAIL_VERIFICATION=false
```
- Monitor email delivery success rate
- Fix any email deliverability issues
- Ensure verification flow is smooth

### Phase 3: Hard Enforcement (Before Public Launch)
```bash
# Block unverified users
REQUIRE_EMAIL_VERIFICATION=true
```
- Enable hard enforcement
- Monitor support tickets
- Be ready to roll back if issues

---

## 📈 Success Metrics

### Measure These:
1. **Email Verification Rate**: % of users who verify within 24 hours
2. **Login Attempt Rate**: % of unverified users trying to login
3. **Resend Rate**: How often users need to resend
4. **Conversion Rate**: % completing full flow (register → verify → login)
5. **Support Tickets**: Volume of verification-related issues
6. **Time to Verification**: Average time from registration to verification

### Target KPIs:
- Verification rate within 24h: > 60%
- Verification rate within 7 days: > 80%
- Resend rate: < 20%
- Support ticket volume: < 5% of registrations

---

## Summary

✅ **Login redirect for unverified users fully specified**  
✅ **Graceful UX with clear next steps**  
✅ **5 new tasks added** (T361-T363, T366-T367)  
✅ **Feature flag controlled** (soft vs hard enforcement)  
✅ **Integrated with email verification** (Phase 15)  
✅ **Complete translations** (EN/ID)  
✅ **Comprehensive testing** (unit, integration, E2E)  

**Implementation Time**: ~5-6 hours
- Backend: 1 hour (T361)
- Frontend: 2.5 hours (T362, T363)
- Testing: 2 hours (T366, T367)

**Status**: 🟢 **FULLY SPECIFIED - READY FOR IMPLEMENTATION**

**Total Project Tasks**: **368 tasks** (from 363)

---

## Before/After Comparison

| Aspect | Before | After |
|--------|--------|-------|
| **Error Message** | Generic "Email not verified" | Dedicated page with guidance |
| **Resend Access** | Have to find resend page | One-click button on same page |
| **Email Context** | User must re-enter email | Email pre-filled from login |
| **Instructions** | None | Step-by-step guide |
| **Help** | No troubleshooting | Spam check, typo tips, support link |
| **Navigation** | Stuck on error | Easy back to login |
| **UX Quality** | Frustrating | Smooth and professional |

**Result**: Significantly better user experience with minimal additional code! 🎉
