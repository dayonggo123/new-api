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

import React, { useState, useEffect, useRef } from 'react';
import { Tabs, Button, Card, Form, Row, Col, Tag, Space, Spin, Typography, Avatar, Popconfirm, SideSheet } from '@douyinfe/semi-ui';
import { IconSave, IconClose, IconPlus, IconBookStroked } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import PromptsPage from '../../components/table/prompts';

const { Text, Title } = Typography;

const CategoryEditModal = ({ visible, onCancel, category, refresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isEdit = category?.id !== undefined;

  const getInitValues = () => ({
    name: '',
    description: '',
    sort_order: 0,
    status: true,
  });

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        formApiRef.current.setValues({
          ...category,
          status: category.status === 1,
        });
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [category?.id, visible]);

  const submit = async (values) => {
    setLoading(true);
    const payload = {
      ...values,
      status: values.status ? 1 : 2,
      sort_order: parseInt(values.sort_order) || 0,
    };
    try {
      let res;
      if (isEdit) {
        res = await API.put(`/api/prompt-category/`, { ...payload, id: category.id });
      } else {
        res = await API.post(`/api/prompt-category/`, payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(isEdit ? t('分类更新成功！') : t('分类创建成功！'));
        refresh();
        onCancel();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  return (
    <SideSheet
      title={
        <Space>
          {isEdit ? (
            <Tag color="blue" shape="circle">{t('更新')}</Tag>
          ) : (
            <Tag color="green" shape="circle">{t('新建')}</Tag>
          )}
          <Title heading={4} className="m-0">
            {isEdit ? t('更新分类信息') : t('创建新的分类')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={500}
      footer={
        <div className="flex justify-end bg-white">
          <Space>
            <Button theme="solid" onClick={() => formApiRef.current?.submitForm()} icon={<IconSave />} loading={loading}>
              {t('提交')}
            </Button>
            <Button theme="light" type="primary" onClick={onCancel} icon={<IconClose />}>
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={onCancel}
    >
      <Spin spinning={loading}>
        <Form initValues={getInitValues()} getFormApi={(api) => (formApiRef.current = api)} onSubmit={submit}>
          {() => (
            <div className="p-4">
              <Card className="!rounded-2xl shadow-sm border-0">
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Input
                      field="name"
                      label={t('名称')}
                      placeholder={t('请输入分类名称')}
                      style={{ width: '100%' }}
                      rules={[{ required: true, message: t('请输入分类名称') }]}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea
                      field="description"
                      label={t('描述')}
                      placeholder={t('请输入分类描述')}
                      rows={2}
                      style={{ width: '100%' }}
                    />
                  </Col>
                  <Col span={12}>
                    <Form.InputNumber
                      field="sort_order"
                      label={t('排序')}
                      placeholder={t('请输入排序值')}
                      min={0}
                      style={{ width: '100%' }}
                    />
                  </Col>
                  <Col span={12}>
                    <div className="flex items-center h-full pt-6">
                      <Form.Switch
                        field="status"
                        label={t('状态')}
                        checkedText={t('启用')}
                        uncheckedText={t('禁用')}
                      />
                    </div>
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

const Prompt = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('prompts');
  const [categories, setCategories] = useState([]);
  const [catLoading, setCatLoading] = useState(false);
  const [editingCategory, setEditingCategory] = useState({ id: undefined });
  const [showCatEdit, setShowCatEdit] = useState(false);

  const loadCategories = async () => {
    setCatLoading(true);
    try {
      const res = await API.get('/api/prompt-category/all');
      const { success, message, data } = res.data;
      if (success) {
        setCategories(data || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setCatLoading(false);
  };

  useEffect(() => {
    loadCategories();
  }, []);

  const handleDeleteCategory = async (id) => {
    try {
      const res = await API.delete(`/api/prompt-category/${id}/`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('操作成功完成！'));
        loadCategories();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const renderCategoryTable = () => {
    return (
      <Card className="!rounded-2xl shadow-sm border-0">
        <div className="flex justify-between items-center mb-4">
          <div className="flex items-center gap-2">
            <Avatar size="small" color="blue">
              <IconBookStroked size={16} />
            </Avatar>
            <Text className="text-lg font-medium">{t('分类列表')}</Text>
          </div>
          <Button
            type="primary"
            size="small"
            icon={<IconPlus />}
            onClick={() => {
              setEditingCategory({ id: undefined });
              setShowCatEdit(true);
            }}
          >
            {t('添加分类')}
          </Button>
        </div>
        <Spin spinning={catLoading}>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b" style={{ borderColor: 'var(--semi-color-border)' }}>
                  <th className="text-left py-2 px-3 font-medium">{t('ID')}</th>
                  <th className="text-left py-2 px-3 font-medium">{t('名称')}</th>
                  <th className="text-left py-2 px-3 font-medium">{t('描述')}</th>
                  <th className="text-left py-2 px-3 font-medium">{t('排序')}</th>
                  <th className="text-left py-2 px-3 font-medium">{t('状态')}</th>
                  <th className="text-right py-2 px-3 font-medium">{t('操作')}</th>
                </tr>
              </thead>
              <tbody>
                {categories.map((cat) => (
                  <tr
                    key={cat.id}
                    className="border-b hover:bg-gray-50"
                    style={{ borderColor: 'var(--semi-color-border)' }}
                  >
                    <td className="py-2 px-3">{cat.id}</td>
                    <td className="py-2 px-3 font-medium">{cat.name}</td>
                    <td className="py-2 px-3">{cat.description || '-'}</td>
                    <td className="py-2 px-3">{cat.sort_order}</td>
                    <td className="py-2 px-3">
                      <Tag color={cat.status === 1 ? 'green' : 'red'} shape="circle">
                        {cat.status === 1 ? t('启用') : t('禁用')}
                      </Tag>
                    </td>
                    <td className="py-2 px-3 text-right">
                      <Space>
                        <Button
                          type="tertiary"
                          size="small"
                          onClick={() => {
                            setEditingCategory(cat);
                            setShowCatEdit(true);
                          }}
                        >
                          {t('编辑')}
                        </Button>
                        <Popconfirm
                          title={t('确定删除此分类吗？')}
                          content={t('此操作不可撤销')}
                          onConfirm={() => handleDeleteCategory(cat.id)}
                        >
                          <Button type="danger" theme="light" size="small">
                            {t('删除')}
                          </Button>
                        </Popconfirm>
                      </Space>
                    </td>
                  </tr>
                ))}
                {categories.length === 0 && (
                  <tr>
                    <td colSpan={6} className="py-8 text-center text-gray-400">
                      {t('暂无数据')}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </Spin>
      </Card>
    );
  };

  return (
    <div className="mt-[60px] px-2">
      <Tabs
        type="line"
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key)}
      >
        <Tabs.TabPane tab={t('提示词管理')} itemKey="prompts">
          <PromptsPage />
        </Tabs.TabPane>
        <Tabs.TabPane tab={t('分类管理')} itemKey="categories">
          {renderCategoryTable()}
        </Tabs.TabPane>
      </Tabs>

      <CategoryEditModal
        visible={showCatEdit}
        onCancel={() => {
          setShowCatEdit(false);
          setEditingCategory({ id: undefined });
        }}
        category={editingCategory}
        refresh={loadCategories}
      />
    </div>
  );
};

export default Prompt;
