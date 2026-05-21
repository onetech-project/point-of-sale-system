/** @type {import('next').NextConfig} */
const fs = require('fs');
const path = require('path');

const noStoreCacheHeader = 'no-cache, no-store, must-revalidate';
const immutableAssetCacheHeader = 'public, max-age=31536000, immutable';
const versionJsonPath = path.join(__dirname, 'public', 'version.json');
const htmlRouteSource = '/((?!api|_next/static|_next/image|assets|version\\.json|.*\\..*).*)';

const readBuildVersion = () => {
  try {
    return JSON.parse(fs.readFileSync(versionJsonPath, 'utf8'));
  } catch {
    return {};
  }
};

const buildVersion = readBuildVersion();
const useStandaloneOutput = process.env.NEXT_OUTPUT === 'standalone';

// Build remote patterns dynamically from environment variables
const buildImageRemotePatterns = () => {
  const patterns = [];

  // Add S3/MinIO endpoint from environment variable
  const s3Endpoint = process.env.NEXT_PUBLIC_S3_ENDPOINT || 'http://localhost:9000';
  try {
    const url = new URL(s3Endpoint);
    patterns.push({
      protocol: url.protocol.replace(':', ''),
      hostname: url.hostname,
      port: url.port || '',
      pathname: '/**',
    });
  } catch (e) {
    console.warn('Invalid NEXT_PUBLIC_S3_ENDPOINT:', s3Endpoint);
  }

  // Add common cloud storage providers
  patterns.push(
    {
      protocol: 'https',
      hostname: '**.amazonaws.com',
      pathname: '/**',
    },
    {
      protocol: 'https',
      hostname: '**.digitaloceanspaces.com',
      pathname: '/**',
    },
    {
      protocol: 'https',
      hostname: '**.r2.cloudflarestorage.com',
      pathname: '/**',
    }
  );

  return patterns;
};

const nextConfig = {
  reactStrictMode: true,
  ...(useStandaloneOutput ? { output: 'standalone' } : {}),
  env: {
    NEXT_PUBLIC_BUILD_HASH: buildVersion.buildHash || '',
    NEXT_PUBLIC_BUILD_TIME: buildVersion.buildTime || '',
  },
  images: {
    remotePatterns: buildImageRemotePatterns(),
    // Disable optimization for external images with query params (presigned URLs)
    unoptimized: true,
  },
  async headers() {
    return [
      {
        source: '/version.json',
        headers: [
          {
            key: 'Cache-Control',
            value: noStoreCacheHeader,
          },
        ],
      },
      {
        source: '/_next/static/:path*',
        headers: [
          {
            key: 'Cache-Control',
            value: immutableAssetCacheHeader,
          },
        ],
      },
      {
        source: htmlRouteSource,
        headers: [
          {
            key: 'Cache-Control',
            value: noStoreCacheHeader,
          },
        ],
      },
    ];
  },
  async rewrites() {
    // Use API_GATEWAY_URL from environment, fallback to localhost
    const apiGatewayUrl = process.env.API_GATEWAY_URL || 'http://localhost:8080';

    return [
      {
        source: '/api/:path*',
        destination: `${apiGatewayUrl}/api/:path*`,
      },
    ];
  },
};

module.exports = nextConfig;
