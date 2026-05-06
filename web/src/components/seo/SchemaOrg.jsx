/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useContext, useMemo } from 'react';
import { StatusContext } from '../../context/Status';

/**
 * Schema.org JSON-LD 结构化数据组件
 * 提升搜索引擎和 AI 引擎对页面内容的理解
 */

/**
 * 组织/软件应用结构化数据（适用于首页、关于页）
 */
export function SoftwareApplicationSchema() {
  const [statusState] = useContext(StatusContext);
  const serverAddress = statusState?.status?.server_address || '';
  const logo = statusState?.status?.logo || '/logo.png';

  const schema = useMemo(() => {
    const logoUrl = logo.startsWith('http') ? logo : `${serverAddress}${logo}`;
    return {
      '@context': 'https://schema.org',
      '@graph': [
        {
          '@type': 'SoftwareApplication',
          name: 'New API',
          applicationCategory: 'DeveloperApplication',
          operatingSystem: 'Any',
          offers: {
            '@type': 'Offer',
            price: '0',
            priceCurrency: 'USD',
          },
          author: {
            '@type': 'Organization',
            name: 'QuantumNous',
            url: 'https://github.com/QuantumNous',
            logo: logoUrl || undefined,
          },
          softwareVersion: statusState?.status?.version || undefined,
          url: serverAddress || undefined,
          description:
            '统一的 AI 模型聚合与分发网关，支持将各类大语言模型跨格式转换为 OpenAI、Claude、Gemini 兼容接口。',
          featureList: [
            '支持 40+ 大模型供应商',
            'OpenAI / Claude / Gemini 格式兼容',
            '统一 API 密钥管理',
            '实时渠道负载均衡',
            '多租户计费与配额管理',
          ],
        },
        {
          '@type': 'Organization',
          name: 'QuantumNous',
          url: 'https://github.com/QuantumNous',
          logo: logoUrl || undefined,
          sameAs: ['https://github.com/QuantumNous/new-api'],
        },
      ],
    };
  }, [serverAddress, logo, statusState?.status?.version]);

  return (
    <script type='application/ld+json'>{JSON.stringify(schema)}</script>
  );
}

/**
 * FAQ 页面结构化数据
 * @param {Array<{question: string, answer: string}>} faqs
 */
export function FAQPageSchema({ faqs = [] }) {
  const [statusState] = useContext(StatusContext);
  const serverAddress = statusState?.status?.server_address || '';

  const schema = useMemo(() => {
    if (!faqs.length) return null;
    return {
      '@context': 'https://schema.org',
      '@type': 'FAQPage',
      mainEntity: faqs.map((faq) => ({
        '@type': 'Question',
        name: faq.question,
        acceptedAnswer: {
          '@type': 'Answer',
          text: faq.answer,
        },
      })),
      url: serverAddress || undefined,
    };
  }, [faqs, serverAddress]);

  if (!schema) return null;
  return <script type='application/ld+json'>{JSON.stringify(schema)}</script>;
}

/**
 * 网页基本结构化数据（适用于任意公开页面）
 * @param {string} pageTitle
 * @param {string} pageDescription
 * @param {string} pathname
 */
export function WebPageSchema({ pageTitle, pageDescription, pathname = '' }) {
  const [statusState] = useContext(StatusContext);
  const serverAddress = statusState?.status?.server_address || '';

  const schema = useMemo(() => {
    const url = serverAddress
      ? `${serverAddress}${pathname.startsWith('/') ? pathname : `/${pathname}`}`
      : undefined;
    return {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: pageTitle,
      description: pageDescription,
      url,
      publisher: {
        '@type': 'Organization',
        name: 'QuantumNous',
        url: 'https://github.com/QuantumNous',
      },
      dateModified: new Date().toISOString(),
    };
  }, [pageTitle, pageDescription, pathname, serverAddress]);

  return <script type='application/ld+json'>{JSON.stringify(schema)}</script>;
}

/**
 * 定价页面结构化数据
 */
export function ProductSchema({ name, description, price, priceCurrency = 'USD' }) {
  const [statusState] = useContext(StatusContext);
  const serverAddress = statusState?.status?.server_address || '';

  const schema = useMemo(() => {
    return {
      '@context': 'https://schema.org',
      '@type': 'Product',
      name,
      description,
      url: serverAddress ? `${serverAddress}/pricing` : undefined,
      brand: {
        '@type': 'Brand',
        name: 'New API',
      },
      offers: price
        ? {
            '@type': 'Offer',
            price,
            priceCurrency,
            availability: 'https://schema.org/InStock',
          }
        : undefined,
    };
  }, [name, description, price, priceCurrency, serverAddress]);

  return <script type='application/ld+json'>{JSON.stringify(schema)}</script>;
}
