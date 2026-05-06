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
import ModelPricingPage from '../../components/table/model-pricing/layout/PricingPage';
import SEO from '../../components/seo/SEO';
import { ProductSchema } from '../../components/seo/SchemaOrg';

const Pricing = () => (
  <>
    <SEO
      title='模型定价'
      description='查看 New API 支持的所有大模型定价信息，包括 OpenAI、Claude、Gemini、DeepSeek 等 40+ 供应商的透明计费标准。'
      pathname='/pricing'
      keywords='AI模型定价, OpenAI价格, Claude价格, Gemini价格, DeepSeek价格, LLM计费, API定价'
    />
    <ProductSchema
      name='New API 模型定价'
      description='查看 New API 支持的所有大模型定价信息，透明计费标准。'
    />
    <ModelPricingPage />
  </>
);

export default Pricing;
