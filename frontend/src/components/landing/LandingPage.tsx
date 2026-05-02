import React from 'react';
import PublicLayout from '@/components/layout/PublicLayout';
import HeroSection from './HeroSection';
import FeaturesSection from './FeaturesSection';
import PricingSection from './PricingSection';
import CtaSection from './CtaSection';

export default function LandingPage() {
  return (
    <PublicLayout>
      <HeroSection />
      <FeaturesSection />
      <PricingSection />
      <CtaSection />
    </PublicLayout>
  );
}
