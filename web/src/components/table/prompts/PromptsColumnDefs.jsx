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
      dataIndex: 'category_id',
      render: (text) => {
        return <div>{getCategoryName(text)}</div>;
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
