import React from 'react';
import Link from 'next/link';

export default function CtaSection() {
  return (
    <section className="py-20 px-4 sm:px-6 lg:px-8 bg-gradient-to-r from-primary-600 to-primary-800">
      <div className="max-w-4xl mx-auto text-center">
        <h2 className="text-4xl md:text-5xl font-bold text-white mb-4">
          Ready to Get Started?
        </h2>
        <p className="text-xl text-primary-100 mb-8 max-w-2xl mx-auto">
          Try Posku free for 7 days — no credit card required. Experience how Posku can transform
          your business operations.
        </p>

        <Link
          href="/signup"
          className="inline-block px-10 py-4 bg-white text-primary-600 font-bold text-lg rounded-lg hover:bg-primary-50 transition-colors shadow-lg hover:shadow-xl"
        >
          Start Free Trial Now
        </Link>

        <p className="text-primary-100 text-sm mt-6">
          Already have an account?{' '}
          <Link href="/login" className="text-white font-semibold hover:underline">
            Sign in here
          </Link>
        </p>
      </div>
    </section>
  );
}
