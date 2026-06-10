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
import { Modal, Tag, Typography } from '@douyinfe/semi-ui';
import { Eye, FileJson, ListOrdered, Lightbulb, MapPin } from 'lucide-react';

const { Text, Title } = Typography;

function parseGeoBlocks(geoBlocksStr) {
  if (!geoBlocksStr || geoBlocksStr === 'null' || geoBlocksStr === '{}') return null;
  try {
    const parsed = JSON.parse(geoBlocksStr);
    if (parsed && typeof parsed === 'object') return parsed;
    return null;
  } catch {
    return null;
  }
}

/**
 * Open a modal to view GEO blocks content
 */
export function openViewGeoBlocksModal(t, record) {
  const geoBlocks = parseGeoBlocks(record.geo_blocks);
  const geoBlocksI18n = record.geo_blocks_i18n;
  const hasGeoBlocks = geoBlocks !== null;
  const hasI18n = geoBlocksI18n && geoBlocksI18n !== '{}' && geoBlocksI18n !== 'null';

  Modal.info({
    title: (
      <div className='flex items-center'>
        <Eye className='mr-2' size={18} />
        {t('GEO 结构化内容')}
        <Tag color='blue' size='small' className='ml-2'>
          {record.title?.slice(0, 20)}
          {record.title?.length > 20 ? '...' : ''}
        </Tag>
      </div>
    ),
    width: 720,
    icon: null,
    content: (
      <div style={{ maxHeight: 520, overflowY: 'auto', paddingRight: 8 }}>
        {!hasGeoBlocks && !hasI18n ? (
          <div className='text-center py-8 text-gray-400'>{t('暂无 GEO 内容')}</div>
        ) : (
          <>
            {/* 中文原文 */}
            {hasGeoBlocks && (
              <div className='mb-5'>
                <Title heading={5} style={{ marginBottom: 12, display: 'flex', alignItems: 'center' }}>
                  <MapPin size={16} className='mr-2' style={{ color: '#16a34a' }} />
                  {t('中文原文')}
                </Title>
                <GeoBlocksContent data={geoBlocks} t={t} />
              </div>
            )}

            {/* 多语言版本 */}
            {hasI18n && (
              <div>
                <Title heading={5} style={{ marginBottom: 12, display: 'flex', alignItems: 'center' }}>
                  <FileJson size={16} className='mr-2' style={{ color: '#2563eb' }} />
                  {t('多语言版本')}
                </Title>
                <pre
                  style={{
                    background: '#f8fafc',
                    border: '1px solid #e2e8f0',
                    borderRadius: 8,
                    padding: 16,
                    fontSize: 13,
                    lineHeight: 1.6,
                    overflowX: 'auto',
                    maxHeight: 280,
                    overflowY: 'auto',
                  }}
                >
                  {JSON.stringify(JSON.parse(geoBlocksI18n), null, 2)}
                </pre>
              </div>
            )}
          </>
        )}
      </div>
    ),
    okText: t('关闭'),
    cancelButtonProps: { style: { display: 'none' } },
  });
}

/**
 * Render parsed GEO blocks content
 */
function GeoBlocksContent({ data, t }) {
  if (!data) return null;

  // 判断是 Prompt GEO (scenarios/steps/tips) 还是 Article GEO (what/why/how/summary)
  const isPromptGeo = data.scenarios !== undefined || data.steps !== undefined || data.tips !== undefined;
  const isArticleGeo = data.what !== undefined || data.why !== undefined || data.how !== undefined || data.summary !== undefined;

  return (
    <div>
      {/* ========== Prompt GEO 结构 ========== */}
      {isPromptGeo && (
        <>
          {/* 适用场景 */}
          {data.scenarios && (
            <div className='mb-4'>
              <div
                style={{
                  padding: 14,
                  background: '#f0fdf4',
                  borderRadius: 8,
                  borderLeft: '4px solid #16a34a',
                }}
              >
                <div className='font-semibold mb-2 flex items-center' style={{ color: '#166534', fontSize: 14 }}>
                  <MapPin size={14} className='mr-1' />
                  {t('适用场景')}
                </div>
                <Text style={{ fontSize: 14, lineHeight: 1.7, color: '#374151' }}>{data.scenarios}</Text>
              </div>
            </div>
          )}

          {/* 使用步骤 */}
          {data.steps && data.steps.length > 0 && (
            <div className='mb-4'>
              <div
                style={{
                  padding: 14,
                  background: '#eff6ff',
                  borderRadius: 8,
                  borderLeft: '4px solid #2563eb',
                }}
              >
                <div className='font-semibold mb-2 flex items-center' style={{ color: '#1e40af', fontSize: 14 }}>
                  <ListOrdered size={14} className='mr-1' />
                  {t('使用步骤')}
                </div>
                <ol style={{ paddingLeft: 20, margin: 0 }}>
                  {data.steps.map((step, idx) => (
                    <li key={idx} style={{ marginBottom: 6, lineHeight: 1.7, fontSize: 14, color: '#374151' }}>
                      {step}
                    </li>
                  ))}
                </ol>
              </div>
            </div>
          )}

          {/* 使用技巧 */}
          {data.tips && (
            <div>
              <div
                style={{
                  padding: 14,
                  background: '#fffbeb',
                  borderRadius: 8,
                  borderLeft: '4px solid #f59e0b',
                }}
              >
                <div className='font-semibold mb-2 flex items-center' style={{ color: '#92400e', fontSize: 14 }}>
                  <Lightbulb size={14} className='mr-1' />
                  {t('使用技巧')}
                </div>
                <Text style={{ fontSize: 14, lineHeight: 1.7, color: '#374151' }}>{data.tips}</Text>
              </div>
            </div>
          )}
        </>
      )}

      {/* ========== Article GEO 结构 ========== */}
      {isArticleGeo && (
        <>
          {/* What - 是什么 */}
          {data.what && (
            <div className='mb-4'>
              <div
                style={{
                  padding: 14,
                  background: '#f0fdf4',
                  borderRadius: 8,
                  borderLeft: '4px solid #16a34a',
                }}
              >
                <div className='font-semibold mb-2 flex items-center' style={{ color: '#166534', fontSize: 14 }}>
                  <MapPin size={14} className='mr-1' />
                  {t('是什么')}
                </div>
                <Text style={{ fontSize: 14, lineHeight: 1.7, color: '#374151' }}>{data.what}</Text>
              </div>
            </div>
          )}

          {/* Why - 为什么 */}
          {data.why && (
            <div className='mb-4'>
              <div
                style={{
                  padding: 14,
                  background: '#eff6ff',
                  borderRadius: 8,
                  borderLeft: '4px solid #2563eb',
                }}
              >
                <div className='font-semibold mb-2 flex items-center' style={{ color: '#1e40af', fontSize: 14 }}>
                  <ListOrdered size={14} className='mr-1' />
                  {t('为什么')}
                </div>
                <Text style={{ fontSize: 14, lineHeight: 1.7, color: '#374151' }}>{data.why}</Text>
              </div>
            </div>
          )}

          {/* How - 怎么做 */}
          {data.how && data.how.length > 0 && (
            <div className='mb-4'>
              <div
                style={{
                  padding: 14,
                  background: '#f5f3ff',
                  borderRadius: 8,
                  borderLeft: '4px solid #7c3aed',
                }}
              >
                <div className='font-semibold mb-2 flex items-center' style={{ color: '#5b21b6', fontSize: 14 }}>
                  <ListOrdered size={14} className='mr-1' />
                  {t('怎么做')}
                </div>
                <ol style={{ paddingLeft: 20, margin: 0 }}>
                  {data.how.map((step, idx) => (
                    <li key={idx} style={{ marginBottom: 6, lineHeight: 1.7, fontSize: 14, color: '#374151' }}>
                      {step}
                    </li>
                  ))}
                </ol>
              </div>
            </div>
          )}

          {/* Summary - 核心总结 */}
          {data.summary && (
            <div>
              <div
                style={{
                  padding: 14,
                  background: '#fffbeb',
                  borderRadius: 8,
                  borderLeft: '4px solid #f59e0b',
                }}
              >
                <div className='font-semibold mb-2 flex items-center' style={{ color: '#92400e', fontSize: 14 }}>
                  <Lightbulb size={14} className='mr-1' />
                  {t('核心总结')}
                </div>
                <Text style={{ fontSize: 14, lineHeight: 1.7, color: '#374151' }}>{data.summary}</Text>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
