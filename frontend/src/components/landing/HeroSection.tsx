'use client';

import React from 'react';
import Link from 'next/link';

export default function HeroSection() {
  const handleLearnMoreClick = () => {
    const featuresSection = document.getElementById('features');
    if (featuresSection) {
      featuresSection.scrollIntoView({ behavior: 'smooth' });
    }
  };

  return (
    <section className="relative py-20 px-4 sm:px-6 lg:px-8 bg-gradient-to-br from-white via-primary-50 to-white">
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
          {/* Left side: Text content */}
          <div className="flex flex-col justify-center">
            <h1 className="text-5xl md:text-6xl font-bold text-gray-900 leading-tight mb-6">
              Manage Your Business Smarter with{' '}
              <span className="text-primary-600">Posku</span>
            </h1>
            <p className="text-xl text-gray-600 mb-4">
              Complete POS solution for restaurants, retail, and cafes
            </p>
            <p className="text-lg text-gray-500 mb-8">
              Start your 7-day free trial today. No credit card required.
            </p>

            {/* CTA Buttons */}
            <div className="flex flex-col sm:flex-row gap-4">
              <Link
                href="/signup"
                className="inline-block px-8 py-4 bg-primary-600 text-white font-semibold rounded-lg hover:bg-primary-700 transition-colors text-center"
              >
                Start Free Trial
              </Link>
              <button
                onClick={handleLearnMoreClick}
                className="inline-block px-8 py-4 bg-white text-primary-600 font-semibold rounded-lg border-2 border-primary-600 hover:bg-primary-50 transition-colors"
              >
                Learn More
              </button>
            </div>

            {/* Trust indicators */}
            <div className="mt-8 flex items-center space-x-3">
              <div className="flex -space-x-2">
                {['A', 'B', 'C'].map((letter, i) => (
                  <div
                    key={i}
                    className="w-8 h-8 rounded-full bg-gradient-to-br from-primary-500 to-primary-600 flex items-center justify-center text-white text-xs font-bold border-2 border-white"
                  >
                    {letter}
                  </div>
                ))}
              </div>
              <p className="text-sm text-gray-600">
                Trusted by 500+ businesses across Indonesia
              </p>
            </div>
          </div>

          {/* Right side: Visual element */}
          <div className="relative">
            <div className="bg-gradient-to-br from-primary-600 to-primary-800 rounded-2xl p-8 shadow-2xl">
              <div className="space-y-6">
                {/* Mock dashboard stats */}
                <div className="bg-white bg-opacity-10 backdrop-blur-sm rounded-lg p-4">
                  <p className="text-white text-sm opacity-75 mb-2">Today's Sales</p>
                  <p className="text-white text-3xl font-bold">Rp 4,852,000</p>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="bg-white bg-opacity-10 backdrop-blur-sm rounded-lg p-4">
                    <p className="text-white text-xs opacity-75 mb-2">Orders</p>
                    <p className="text-white text-2xl font-bold">34</p>
                  </div>
                  <div className="bg-white bg-opacity-10 backdrop-blur-sm rounded-lg p-4">
                    <p className="text-white text-xs opacity-75 mb-2">Customers</p>
                    <p className="text-white text-2xl font-bold">28</p>
                  </div>
                </div>

                <div className="bg-white bg-opacity-10 backdrop-blur-sm rounded-lg p-4">
                  <p className="text-white text-sm opacity-75 mb-3">Top Products</p>
                  <div className="space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-white text-sm">Iced Latte</span>
                      <span className="text-white font-semibold">12x</span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-white text-sm">Espresso</span>
                      <span className="text-white font-semibold">8x</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Decorative elements */}
            <div className="absolute -top-4 -right-4 w-24 h-24 bg-primary-100 rounded-full opacity-50"></div>
            <div className="absolute -bottom-4 -left-4 w-32 h-32 bg-primary-100 rounded-full opacity-30"></div>
          </div>
        </div>
      </div>
    </section>
  );
}
