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
import { Helmet } from 'react-helmet-async';
import { StatusContext } from '../../context/Status';
import { useTranslation } from 'react-i18next';

/**
 * SEO 组件 —— 统一管理页面的 title, meta, Open Graph, Twitter Card 和 canonical
 *
 * @param {string} title        — 页面标题（不含站点后缀）
 * @param {string} description  — 页面描述
 * @param {string} pathname     — 当前路由 path，用于拼接 canonical 和 og:url
 * @param {string} ogImage      — Open Graph 图片地址（默认使用站点 logo）
 * @param {boolean} noindex     — 是否禁止搜索引擎索引
 * @param {string} keywords     — 页面关键词，逗号分隔
 * @param {string} type         — og:type，默认 website
 */
export default function SEO({
  title,
  description,
  pathname = '',
  ogImage,
  noindex = false,
  keywords,
  type = 'website',
}) {
  const { i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);

  const siteName = statusState?.status?.system_name || 'New API';
  const serverAddress = statusState?.status?.server_address || '';
  const logo = statusState?.status?.logo || '/logo.png';
  const lang = i18n.language || 'zh';

  const fullTitle = useMemo(() => {
    if (!title) return siteName;
    return `${title} | ${siteName}`;
  }, [title, siteName]);

  const canonicalUrl = useMemo(() => {
    const base = serverAddress || (typeof window !== 'undefined' ? window.location.origin : '');
    if (!base) return '';
    const path = pathname.startsWith('/') ? pathname : `/${pathname}`;
    return `${base}${path}`;
  }, [serverAddress, pathname]);

  const ogImageUrl = useMemo(() => {
    if (ogImage) return ogImage;
    if (logo.startsWith('http')) return logo;
    const base = serverAddress || (typeof window !== 'undefined' ? window.location.origin : '');
    return `${base}${logo}`;
  }, [ogImage, logo, serverAddress]);

  return (
    <Helmet>
      <html lang={lang.startsWith('zh') ? 'zh-CN' : lang} />
      <title>{fullTitle}</title>
      {description && <meta name='description' content={description} />}
      {keywords && <meta name='keywords' content={keywords} />}

      {/* Canonical */}
      {canonicalUrl && <link rel='canonical' href={canonicalUrl} />}

      {/* Robots */}
      {noindex ? (
        <meta name='robots' content='noindex, nofollow' />
      ) : (
        <meta name='robots' content='index, follow' />
      )}

      {/* Open Graph */}
      <meta property='og:title' content={fullTitle} />
      {description && <meta property='og:description' content={description} />}
      <meta property='og:site_name' content={siteName} />
      {canonicalUrl && <meta property='og:url' content={canonicalUrl} />}
      <meta property='og:type' content={type} />
      <meta property='og:locale' content={lang.startsWith('zh') ? 'zh_CN' : 'en_US'} />
      {ogImageUrl && <meta property='og:image' content={ogImageUrl} />}

      {/* Twitter Card */}
      <meta name='twitter:card' content='summary_large_image' />
      <meta name='twitter:title' content={fullTitle} />
      {description && <meta name='twitter:description' content={description} />}
      {ogImageUrl && <meta name='twitter:image' content={ogImageUrl} />}

      {/* GEO / AI Search optimization hints */}
      <meta name='author' content='QuantumNous' />
      <meta name='copyright' content={`© ${new Date().getFullYear()} QuantumNous`} />
    </Helmet>
  );
}
