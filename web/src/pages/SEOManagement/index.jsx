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

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Row,
  Col,
  Tag,
  Space,
  Spin,
  Typography,
  SideSheet,
  Pagination,
  Avatar,
  Empty,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconClose,
  IconSearch,
  IconRefresh,
  IconEdit,
  IconBookStroked,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Text, Title } = Typography;

const SEOEditModal = ({ visible, onCancel, promptId, refresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const formApiRef = useRef(null);
  const [data, setData] = useState({
    title: '',
    category_name: '',
    seo_keywords: '',
    intro: '',
    faq: '',
  });

  const loadDetail = async () => {
    if (!promptId) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/prompt/seo/${promptId}`);
      const { success, message, data } = res.data;
      if (success) {
        const values = {
          ...data,
          faq: data.faq || '',
        };
        setData(values);
        formApiRef.current?.setValues(values);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (visible && promptId) {
      loadDetail();
    }
  }, [visible, promptId]);

  const submit = async (values) => {
    setSaving(true);
    try {
      const res = await API.put(`/api/prompt/seo/${promptId}`, {
        id: promptId,
        seo_keywords: values.seo_keywords || '',
        intro: values.intro || '',
        faq: values.faq || '',
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('SEO 内容更新成功'));
        refresh();
        onCancel();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setSaving(false);
  };

  return (
    <SideSheet
      title={
        <Space>
          <Tag color='blue' shape='circle'>
            {t('编辑')}
          </Tag>
          <Title heading={4} className='m-0'>
            {t('编辑 SEO 内容')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={600}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button
              theme='solid'
              onClick={() => formApiRef.current?.submitForm()}
              icon={<IconSave />}
              loading={saving}
            >
              {t('保存')}
            </Button>
            <Button
              theme='light'
              type='primary'
              onClick={onCancel}
              icon={<IconClose />}
            >
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={onCancel}
    >
      <Spin spinning={loading}>
        <Form
          initValues={data}
          getFormApi={(api) => (formApiRef.current = api)}
          onSubmit={submit}
        >
          {() => (
            <div className='p-4'>
              <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
                <div className='flex items-center mb-3'>
                  <Avatar size='small' color='blue' className='mr-2 shadow-md'>
                    <IconBookStroked size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>
                      {data.title || t('提示词信息')}
                    </Text>
                    {data.category_name && (
                      <div className='text-xs text-gray-600'>
                        {t('分类')}: {data.category_name}
                      </div>
                    )}
                  </div>
                </div>
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.TextArea
                      field='seo_keywords'
                      label={t('SEO 关键词')}
                      placeholder={t('输入 SEO 关键词，用逗号分隔')}
                      rows={2}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea
                      field='intro'
                      label={t('介绍文案')}
                      placeholder={t('输入介绍文案，300字以内')}
                      rows={4}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea
                      field='faq'
                      label={t('FAQ (JSON 格式)')}
                      placeholder={t('[{"question":"...","answer":"..."}]')}
                      rows={6}
                      showClear
                    />
                  </Col>
                </Row>
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

const SEOManagement = () => {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [editingId, setEditingId] = useState(null);
  const [showEdit, setShowEdit] = useState(false);
  const [regenerating, setRegenerating] = useState({});

  const loadData = useCallback(
    async (page = activePage, size = pageSize, search = keyword) => {
      setLoading(true);
      try {
        let url = `/api/prompt/seo/list?p=${page}&page_size=${size}`;
        if (search) {
          url += `&keyword=${encodeURIComponent(search)}`;
        }
        const res = await API.get(url);
        const { success, message, data } = res.data;
        if (success) {
          setItems(data.items || []);
          setActivePage(data.page <= 0 ? 1 : data.page);
          setTotal(data.total || 0);
        } else {
          showError(message);
        }
      } catch (error) {
        showError(error.message);
      }
      setLoading(false);
    },
    [activePage, pageSize, keyword]
  );

  useEffect(() => {
    loadData(1, pageSize, '');
  }, [pageSize]);

  const handleSearch = () => {
    setActivePage(1);
    loadData(1, pageSize, keyword);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    loadData(page, pageSize, keyword);
  };

  const handleRegenerate = async (id) => {
    setRegenerating((prev) => ({ ...prev, [id]: true }));
    try {
      const res = await API.post(`/api/prompt/seo/${id}/regenerate`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('AI 重新生成已启动'));
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setRegenerating((prev) => ({ ...prev, [id]: false }));
  };

  const openEdit = (id) => {
    setEditingId(id);
    setShowEdit(true);
  };

  const closeEdit = () => {
    setShowEdit(false);
    setEditingId(null);
  };

  const truncate = (text, maxLen = 60) => {
    if (!text) return '-';
    return text.length > maxLen ? text.slice(0, maxLen) + '...' : text;
  };

  const renderFaqCount = (faqStr) => {
    if (!faqStr) return <Tag size='small'>{t('无')}</Tag>;
    try {
      const arr = JSON.parse(faqStr);
      const count = Array.isArray(arr) ? arr.length : 0;
      if (count === 0) return <Tag size='small'>{t('无')}</Tag>;
      return (
        <Tag color='blue' size='small'>
          {count} {t('条')}
        </Tag>
      );
    } catch {
      return (
        <Tag size='small' type='danger'>
          {t('格式错误')}
        </Tag>
      );
    }
  };

  const hasSEO = (item) => {
    return !!(item.seo_keywords || item.intro);
  };

  return (
    <div className='mt-[60px] px-2'>
      <Card className='!rounded-2xl shadow-sm border-0 mb-4'>
        <div className='flex flex-wrap items-center justify-between gap-3 mb-4'>
          <div className='flex items-center gap-2'>
            <Avatar size='small' color='green'>
              <IconBookStroked size={16} />
            </Avatar>
            <Text className='text-lg font-medium'>{t('SEO 管理')}</Text>
          </div>
          <div className='flex items-center gap-2'>
            <Input
              placeholder={t('搜索提示词标题')}
              value={keyword}
              onChange={(v) => setKeyword(v)}
              onEnterPress={handleSearch}
              showClear
              suffix={
                <Button
                  type='tertiary'
                  size='small'
                  icon={<IconSearch />}
                  onClick={handleSearch}
                />
              }
            />
          </div>
        </div>

        <Spin spinning={loading}>
          <div className='overflow-x-auto'>
            <table className='w-full text-sm'>
              <thead>
                <tr
                  className='border-b'
                  style={{ borderColor: 'var(--semi-color-border)' }}
                >
                  <th className='text-left py-2 px-3 font-medium w-16'>
                    {t('ID')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('标题')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('分类')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('SEO 关键词')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium'>
                    {t('介绍文案')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium w-20'>
                    {t('FAQ')}
                  </th>
                  <th className='text-left py-2 px-3 font-medium w-24'>
                    {t('状态')}
                  </th>
                  <th className='text-right py-2 px-3 font-medium w-40'>
                    {t('操作')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr
                    key={item.id}
                    className='border-b hover:bg-gray-50'
                    style={{ borderColor: 'var(--semi-color-border)' }}
                  >
                    <td className='py-2 px-3'>{item.id}</td>
                    <td className='py-2 px-3 font-medium'>
                      <div className='flex items-center gap-1'>
                        {item.title}
                        {hasSEO(item) && (
                          <Tag color='green' size='small'>
                            SEO
                          </Tag>
                        )}
                      </div>
                    </td>
                    <td className='py-2 px-3'>
                      {item.category_name || '-'}
                    </td>
                    <td className='py-2 px-3 text-gray-600'>
                      {truncate(item.seo_keywords, 40)}
                    </td>
                    <td className='py-2 px-3 text-gray-600'>
                      {truncate(item.intro, 50)}
                    </td>
                    <td className='py-2 px-3'>
                      {renderFaqCount(item.faq)}
                    </td>
                    <td className='py-2 px-3'>
                      <Tag
                        color={item.status === 1 ? 'green' : 'red'}
                        shape='circle'
                        size='small'
                      >
                        {item.status === 1 ? t('启用') : t('禁用')}
                      </Tag>
                    </td>
                    <td className='py-2 px-3 text-right'>
                      <Space>
                        <Button
                          type='tertiary'
                          size='small'
                          icon={<IconEdit />}
                          onClick={() => openEdit(item.id)}
                        >
                          {t('编辑')}
                        </Button>
                        <Button
                          type='tertiary'
                          size='small'
                          icon={<IconRefresh />}
                          loading={regenerating[item.id]}
                          onClick={() => handleRegenerate(item.id)}
                        >
                          {t('重新生成')}
                        </Button>
                      </Space>
                    </td>
                  </tr>
                ))}
                {items.length === 0 && (
                  <tr>
                    <td
                      colSpan={8}
                      className='py-8 text-center text-gray-400'
                    >
                      <Empty description={t('暂无数据')} />
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {total > 0 && (
            <div className='flex justify-end mt-4'>
              <Pagination
                currentPage={activePage}
                pageSize={pageSize}
                total={total}
                showSizeChanger
                pageSizeOpts={[10, 20, 50, 100]}
                onPageChange={handlePageChange}
                onPageSizeChange={(size) => {
                  setPageSize(size);
                  setActivePage(1);
                }}
              />
            </div>
          )}
        </Spin>
      </Card>

      <SEOEditModal
        visible={showEdit}
        onCancel={closeEdit}
        promptId={editingId}
        refresh={() => loadData(activePage, pageSize, keyword)}
      />
    </div>
  );
};

export default SEOManagement;
