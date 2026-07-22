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

import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Table,
  Modal,
  Form,
  Input,
  InputNumber,
  Switch,
  Tag,
  Space,
  Popconfirm,
  Typography,
  Col,
  Row,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsTikHubPrices(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const formRef = useRef();

  const columns = [
    {
      title: t('接口标识'),
      dataIndex: 'endpoint',
      key: 'endpoint',
      width: 150,
    },
    {
      title: t('接口名称'),
      dataIndex: 'name',
      key: 'name',
      width: 180,
    },
    {
      title: t('普通价格'),
      key: 'price',
      width: 100,
      render: (_, record) => (
        <Space vertical spacing={4} style={{ fontSize: 12 }}>
          <Text>${record.price?.toFixed(4) || '0.0000'}</Text>
          <Text type="secondary">免费: {record.free_quota || 0}</Text>
        </Space>
      ),
    },
    {
      title: t('VIP价格'),
      key: 'vip_price',
      width: 100,
      render: (_, record) => (
        <Space vertical spacing={4} style={{ fontSize: 12 }}>
          <Text>${record.vip_price?.toFixed(4) || '0.0000'}</Text>
          <Text type="secondary">免费: {record.vip_free_quota || 0}</Text>
        </Space>
      ),
    },
    {
      title: t('SVIP价格'),
      key: 'svip_price',
      width: 100,
      render: (_, record) => (
        <Space vertical spacing={4} style={{ fontSize: 12 }}>
          <Text>${record.svip_price?.toFixed(4) || '0.0000'}</Text>
          <Text type="secondary">免费: {record.svip_free_quota || 0}</Text>
        </Space>
      ),
    },
    {
      title: t('启用'),
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled) => (
        <Tag color={enabled ? 'green' : 'grey'}>
          {enabled ? t('是') : t('否')}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      width: 150,
      render: (text, record) => (
        <Space>
          <Button type="primary" size="small" onClick={() => handleEdit(record)}>
            {t('编辑')}
          </Button>
          <Popconfirm
            title={t('确定删除此配置？')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('确定')}
            cancelText={t('取消')}
          >
            <Button type="danger" size="small">
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/tikhub/prices');
      if (res.data.success) {
        setData(res.data.data || []);
      } else {
        showError(res.data.message || t('获取数据失败'));
      }
    } catch (error) {
      showError(t('获取数据失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleEdit = (record) => {
    setEditingItem(record);
    setTimeout(() => {
      formRef.current.setValues({
        name: record.name,
        description: record.description,
        price: record.price || 0,
        vip_price: record.vip_price || 0,
        svip_price: record.svip_price || 0,
        free_quota: record.free_quota || 0,
        vip_free_quota: record.vip_free_quota || 0,
        svip_free_quota: record.svip_free_quota || 0,
        enabled: record.enabled,
      });
    }, 0);
    setModalVisible(true);
  };

  const handleAdd = () => {
    setEditingItem(null);
    setTimeout(() => {
      formRef.current.setValues({
        endpoint: '',
        name: '',
        description: '',
        price: 0.01,
        vip_price: 0,
        svip_price: 0,
        free_quota: 0,
        vip_free_quota: 0,
        svip_free_quota: 0,
        enabled: true,
      });
    }, 0);
    setModalVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = formRef.current.getValues();
      let res;
      if (editingItem) {
        res = await API.put(`/api/admin/tikhub/prices/${editingItem.id}`, values);
      } else {
        res = await API.post('/api/admin/tikhub/prices', values);
      }
      if (res.data.success) {
        showSuccess(t('保存成功'));
        setModalVisible(false);
        fetchData();
      } else {
        showError(res.data.message || t('保存失败'));
      }
    } catch (error) {
      showError(t('保存失败'));
    }
  };

  const handleDelete = async (id) => {
    try {
      const res = await API.delete(`/api/admin/tikhub/prices/${id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        fetchData();
      } else {
        showError(res.data.message || t('删除失败'));
      }
    } catch (error) {
      showError(t('删除失败'));
    }
  };

  const handleInit = async () => {
    try {
      const res = await API.post('/api/admin/tikhub/prices/init');
      if (res.data.success) {
        showSuccess(t('初始化成功'));
        fetchData();
      } else {
        showError(res.data.message || t('初始化失败'));
      }
    } catch (error) {
      showError(t('初始化失败'));
    }
  };

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" onClick={handleAdd}>
            {t('新增配置')}
          </Button>
          <Button onClick={handleInit}>
            {t('初始化默认配置')}
          </Button>
        </Space>
      </div>
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        rowKey="id"
        pagination={false}
      />
      <Modal
        title={editingItem ? t('编辑配置') : t('新增配置')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        okText={t('保存')}
        cancelText={t('取消')}
        width={600}
      >
        <Form getFormApi={(formAPI) => (formRef.current = formAPI)}>
          {!editingItem && (
            <Form.Input
              field="endpoint"
              label={t('接口标识')}
              placeholder={t('例如: video')}
              rules={[{ required: true, message: t('请输入接口标识') }]}
            />
          )}
          <Form.Input
            field="name"
            label={t('接口名称')}
            placeholder={t('例如: 获取单个视频数据')}
            rules={[{ required: true, message: t('请输入接口名称') }]}
          />
          <Form.Input
            field="description"
            label={t('描述')}
            placeholder={t('接口描述')}
          />

          <div style={{ margin: '16px 0', borderTop: '1px solid var(--semi-color-border)' }}>
            <Text strong style={{ display: 'block', margin: '12px 0' }}>{t('普通用户')}</Text>
            <Row gutter={16}>
              <Col span={12}>
                <Form.InputNumber
                  field="price"
                  label={t('价格 (USD)')}
                  min={0}
                  step={0.01}
                  precision={4}
                  placeholder={t('例如: 0.01')}
                />
              </Col>
              <Col span={12}>
                <Form.InputNumber
                  field="free_quota"
                  label={t('免费条数')}
                  min={0}
                  step={1}
                  placeholder={t('免费调用次数')}
                />
              </Col>
            </Row>
          </div>

          <div style={{ margin: '16px 0', borderTop: '1px solid var(--semi-color-border)' }}>
            <Text strong style={{ display: 'block', margin: '12px 0' }}>{t('VIP 用户')}</Text>
            <Row gutter={16}>
              <Col span={12}>
                <Form.InputNumber
                  field="vip_price"
                  label={t('价格 (USD)')}
                  min={0}
                  step={0.01}
                  precision={4}
                  placeholder={t('VIP 价格')}
                />
              </Col>
              <Col span={12}>
                <Form.InputNumber
                  field="vip_free_quota"
                  label={t('免费条数')}
                  min={0}
                  step={1}
                  placeholder={t('免费调用次数')}
                />
              </Col>
            </Row>
          </div>

          <div style={{ margin: '16px 0', borderTop: '1px solid var(--semi-color-border)' }}>
            <Text strong style={{ display: 'block', margin: '12px 0' }}>{t('SVIP 用户')}</Text>
            <Row gutter={16}>
              <Col span={12}>
                <Form.InputNumber
                  field="svip_price"
                  label={t('价格 (USD)')}
                  min={0}
                  step={0.01}
                  precision={4}
                  placeholder={t('SVIP 价格')}
                />
              </Col>
              <Col span={12}>
                <Form.InputNumber
                  field="svip_free_quota"
                  label={t('免费条数')}
                  min={0}
                  step={1}
                  placeholder={t('免费调用次数')}
                />
              </Col>
            </Row>
          </div>

          <Form.Switch
            field="enabled"
            label={t('启用收费')}
            checkedText={t('是')}
            uncheckedText={t('否')}
          />
        </Form>
      </Modal>
    </>
  );
}
