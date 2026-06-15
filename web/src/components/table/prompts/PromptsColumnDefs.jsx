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
import { Tag, Button, Space, Popconfirm } from '@douyinfe/semi-ui';
import {
  PROMPT_STATUS_MAP,
} from '../../../constants/prompt.constants';
import { getLucideIcon } from '../../../helpers/render';
import { openViewGeoBlocksModal } from '../../../components/modals/ViewGeoBlocksModal';
import { deriveScenes } from '../../../helpers/scene-derive';

/**
 * Render prompt status
 */
const renderStatus = (status, t) => {
  const statusConfig = PROMPT_STATUS_MAP[status];
  if (statusConfig) {
    return (
      <Tag color={statusConfig.color} shape='circle'>
        {t(statusConfig.text)}
      </Tag>
    );
  }
  return (
    <Tag color='black' shape='circle'>
      {t('未知状态')}
    </Tag>
  );
};

/**
 * Format timestamp (seconds) to readable date string
 */
const formatTime = (ts) => {
  if (!ts || ts === 0) return '-';
  const d = new Date(ts * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

/**
 * Parse partial translation error like "partial: en,fr" into missing languages array
 */
const getPartialError = (error) => {
  if (!error || typeof error !== 'string') return null;
  const match = error.match(/^partial:\s*(.+)$/i);
  if (!match) return null;
  const langs = match[1].split(',').map((l) => l.trim()).filter(Boolean);
  return langs.length > 0 ? { langs } : null;
};

/**
 * Calculate translation progress from title_i18n / i18n / content_en
 * Returns "X/11"
 */
const getTranslationProgress = (record) => {
  const targetLangs = ['en', 'fr', 'ru', 'ja', 'vi', 'ko', 'es', 'de', 'pt', 'it', 'ar'];
  let titleMap = {};
  let contentMap = {};
  try {
    if (record.title_i18n) titleMap = JSON.parse(record.title_i18n);
  } catch (e) { /* ignore */ }
  try {
    if (record.i18n) contentMap = JSON.parse(record.i18n);
  } catch (e) { /* ignore */ }

  let completed = 0;
  for (const lang of targetLangs) {
    const hasTitle = titleMap[lang] && String(titleMap[lang]).trim() !== '';
    let hasContent = contentMap[lang] && String(contentMap[lang]).trim() !== '';
    if (lang === 'en' && record.content_en && String(record.content_en).trim() !== '') {
      hasContent = true;
    }
    if (hasTitle && hasContent) completed++;
  }
  return `${completed}/11`;
};

/**
 * Get prompts table column definitions
 */
export const getPromptsColumns = ({
  t,
  categories,
  setEditingPrompt,
  setShowEdit,
  deletePrompt,
}) => {
  const getCategoryName = (categoryId) => {
    const cat = categories.find((c) => c.id === categoryId);
    return cat ? cat.name : '-';
  };

  return [
    {
      title: t('ID'),
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('标题'),
      dataIndex: 'title',
      render: (text) => {
        return <div className='font-medium'>{text}</div>;
      },
    },
    {
      title: t('分类'),
      dataIndex: 'category_name',
      render: (text) => {
        return <div>{text || '-'}</div>;
      },
    },
    {
      title: t('业务场景'),
      dataIndex: 'scenes',
      width: 180,
      render: (_text, record) => {
        const scenes = deriveScenes(record);
        if (scenes.length === 0) {
          return <span className='text-gray-400 text-xs'>-</span>;
        }
        return (
          <Space wrap>
            {scenes.map((s) => (
              <Tag key={s.scene} color={s.color} size='small' title={s.scene}>
                {s.icon} {s.scene}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: t('类型'),
      dataIndex: 'media_type',
      width: 80,
      render: (text) => {
        return (
          <Tag color={text === 'video' ? 'purple' : 'blue'} size='small'>
            {text === 'video' ? t('视频') : t('图片')}
          </Tag>
        );
      },
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (text) => {
        return <div>{renderStatus(text, t)}</div>;
      },
    },
    {
      title: t('已翻译'),
      dataIndex: 'is_translated',
      width: 90,
      render: (text, record) => {
        const partialError = getPartialError(record.translation_error);
        if (text) {
          return (
            <Tag color='green' size='small'>
              {t('是')}
            </Tag>
          );
        }
        if (partialError) {
          return (
            <Tag color='orange' size='small' title={`${t('缺失语言')}: ${partialError.langs.join(', ')}`}>
              {t('部分失败')}
            </Tag>
          );
        }
        if (record.translation_error) {
          return (
            <Tag color='red' size='small' title={record.translation_error}>
              {t('失败')}
            </Tag>
          );
        }
        return (
          <Tag color='grey' size='small'>
            {t('否')}
          </Tag>
        );
      },
    },
    {
      title: t('翻译进度'),
      dataIndex: 'translation_progress',
      width: 100,
      render: (_text, record) => {
        const progress = getTranslationProgress(record);
        const [completed] = progress.split('/').map(Number);
        const isDone = completed === 11;
        const partialError = getPartialError(record.translation_error);
        const hasRealError = !!record.translation_error && !partialError;
        return (
          <Tag
            color={hasRealError ? 'red' : partialError ? 'orange' : isDone ? 'green' : 'blue'}
            size='small'
            title={partialError ? `${t('缺失语言')}: ${partialError.langs.join(', ')}` : record.translation_error}
          >
            {progress}
          </Tag>
        );
      },
    },
    {
      title: t('GEO 结构'),
      dataIndex: 'geo_blocks',
      width: 100,
      render: (text, record) => {
        const has = text && text !== '{}' && text !== 'null';
        return (
          <Tag
            color={has ? 'green' : 'red'}
            size='small'
            style={has ? { cursor: 'pointer' } : {}}
            onClick={has ? () => openViewGeoBlocksModal(t, record) : undefined}
          >
            {has ? t('已生成') : t('未生成')}
          </Tag>
        );
      },
    },
    {
      title: t('排序'),
      dataIndex: 'sort_order',
      width: 80,
    },
    {
      title: t('使用次数'),
      dataIndex: 'usage_count',
      width: 100,
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      width: 160,
      render: (text) => (
        <span style={{ color: 'var(--semi-color-text-2)', fontSize: 13 }}>
          {formatTime(text)}
        </span>
      ),
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      width: 150,
      render: (text, record) => {
        return (
          <Space>
            <Button
              type='tertiary'
              size='small'
              icon={getLucideIcon('detail')}
              onClick={() => {
                setEditingPrompt(record);
                setShowEdit(true);
              }}
            >
              {t('编辑')}
            </Button>
            <Popconfirm
              title={t('确定删除此提示词吗？')}
              content={t('此操作不可撤销')}
              onConfirm={() => {
                deletePrompt(record.id);
              }}
            >
              <Button
                type='danger'
                theme='light'
                size='small'
                icon={getLucideIcon('setting')}
              >
                {t('删除')}
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];
};
