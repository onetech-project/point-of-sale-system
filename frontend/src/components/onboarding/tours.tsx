'use client';

import React from 'react';
import type { OnbordaProps } from 'onborda';
import { ROLES, type Role } from '@/constants/roles';

export const ONBOARDING_TOUR_VERSION = 1;

export const ROLE_TOUR_KEYS: Record<Role, string | null> = {
  [ROLES.OWNER]: 'owner-onboarding',
  [ROLES.MANAGER]: 'manager-onboarding',
  [ROLES.CASHIER]: 'cashier-onboarding',
  [ROLES.ADMIN]: null,
};

export function getTourKeyForRole(role?: Role | string | null): string | null {
  if (!role) return null;
  return ROLE_TOUR_KEYS[role as Role] ?? null;
}

export const roleOnboardingTours: OnbordaProps['steps'] = [
  {
    tour: 'owner-onboarding',
    steps: [
      {
        icon: null,
        title: 'Business dashboard',
        content: <>Track revenue, orders, inventory value, and operational work from one place.</>,
        selector: '#onboarding-business-metrics',
        side: 'bottom',
        pointerPadding: 10,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Team access',
        content: <>Invite managers and cashiers, then adjust roles as your store grows.</>,
        selector: '#sidebar-nav-team',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Products and stock',
        content: <>Keep the catalog, categories, and stock levels ready for orders.</>,
        selector: '#sidebar-nav-products',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Order operations',
        content: <>Open delayed orders and daily order work from the shared order surface.</>,
        selector: '#onboarding-operational-tasks',
        side: 'top',
        pointerPadding: 10,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Menu QR',
        content: <>Generate a QR code so customers can open your public menu quickly.</>,
        selector: '#sidebar-nav-qr-generator',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
    ],
  },
  {
    tour: 'manager-onboarding',
    steps: [
      {
        icon: null,
        title: 'Manager overview',
        content: <>Review performance trends and the work that needs attention today.</>,
        selector: '#onboarding-business-metrics',
        side: 'bottom',
        pointerPadding: 10,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Catalog tools',
        content: <>Use products and inventory tools to keep shelves and menus accurate.</>,
        selector: '#sidebar-nav-products',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Team coordination',
        content: <>Invite team members and manage day-to-day staff access.</>,
        selector: '#sidebar-nav-team',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Operational tasks',
        content: <>Follow delayed orders and low-stock alerts before they affect service.</>,
        selector: '#onboarding-operational-tasks',
        side: 'top',
        pointerPadding: 10,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Menu QR',
        content: <>Use the QR generator when the store needs a fresh public menu code.</>,
        selector: '#sidebar-nav-qr-generator',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
    ],
  },
  {
    tour: 'cashier-onboarding',
    steps: [
      {
        icon: null,
        title: 'Daily work',
        content: <>Start here to see delayed orders and restock tasks that need attention.</>,
        selector: '#onboarding-operational-tasks',
        side: 'bottom',
        pointerPadding: 10,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Orders',
        content: <>Open the order list to review customer and offline order details.</>,
        selector: '#sidebar-nav-orders',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Menu QR',
        content: <>Open the QR generator when a customer needs the public menu code.</>,
        selector: '#sidebar-nav-qr-generator',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
      {
        icon: null,
        title: 'Your profile',
        content: <>Keep your account details current from the profile page.</>,
        selector: '#sidebar-nav-profile',
        side: 'right',
        pointerPadding: 8,
        pointerRadius: 8,
      },
    ],
  },
];
