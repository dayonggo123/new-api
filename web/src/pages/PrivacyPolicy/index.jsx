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

import React from 'react';
import { useTranslation } from 'react-i18next';
import DocumentRenderer from '../../components/common/DocumentRenderer';
import SEO from '../../components/seo/SEO';

const PrivacyPolicy = () => {
  const { t } = useTranslation();

  return (
    <>
      <SEO
        title={t('隐私政策')}
        description={t('New API 隐私政策 —— 了解我们如何收集、使用和保护您的个人信息。')}
        pathname='/privacy-policy'
      />
      <DocumentRenderer
        apiEndpoint='/api/privacy-policy'
        title={t('隐私政策')}
        cacheKey='privacy_policy'
        emptyMessage={t('加载隐私政策内容失败...')}
      />
    </>
  );
};

export default PrivacyPolicy;
