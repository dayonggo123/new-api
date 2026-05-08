import React, { useState, useEffect, useCallback } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Table,
  Tag,
  Space,
  Modal,
  Popconfirm,
  Typography,
  Empty,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconRefresh,
  IconDelete,
  IconEdit,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Title } = Typography;

const colorOptions = [
  'blue', 'green', 'orange', 'red', 'purple', 'pink', 'cyan', 'yellow',
];

export default function TierManagement() {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTier, setEditingTier] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [formApi, setFormApi] = useState(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/tiers');
      const { success, data: result } = res.data;
      if (success) {
        setData(result || []);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const payload = editingTier ? { ...values, id: editingTier.id } : values;
      const url = '/api/admin/tiers';
      const method = editingTier ? 'put' : 'post';
      const res = await API[method](url, payload);
      const { success, message } = res.data;
      if (success) {
        showSuccess(message || (editingTier ? '更新成功' : '创建成功'));
        setModalVisible(false);
        setEditingTier(null);
        formApi?.reset();
        loadData();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setSubmitting(false);
  };

  const handleDelete = async (id) => {
    try {
      const res = await API.delete(`/api/admin/tiers/${id}`);
      if (res.data.success) {
        showSuccess('删除成功');
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
  };

  const openEdit = (tier) => {
    setEditingTier(tier);
    setModalVisible(true);
    setTimeout(() => {
      formApi?.setValues(tier);
    }, 0);
  };

  const openCreate = () => {
    setEditingTier(null);
    setModalVisible(true);
    setTimeout(() => {
      formApi?.reset();
    }, 0);
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '层级',
      dataIndex: 'level',
      width: 80,
    },
    {
      title: '名称',
      dataIndex: 'name',
      render: (name, record) => (
        <Tag color={record.color || 'blue'}>{name}</Tag>
      ),
    },
    {
      title: '最低积分',
      dataIndex: 'min_points',
      width: 100,
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
    },
    {
      title: '操作',
      width: 120,
      render: (_, record) => (
        <Space>
          <Button
            icon={<IconEdit />}
            theme='light'
            onClick={() => openEdit(record)}
          />
          <Popconfirm
            title='确认删除'
            content={`确定要删除层级 "${record.name}" 吗？`}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button icon={<IconDelete />} theme='light' type='danger' />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Title heading={4} style={{ margin: 0 }}>用户层级管理</Title>
          </div>
        }
        headerExtraContent={
          <Space>
            <Button icon={<IconRefresh />} onClick={loadData} loading={loading}>
              刷新
            </Button>
            <Button type='primary' icon={<IconPlus />} onClick={openCreate}>
              新建层级
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          empty={<Empty description='暂无层级' />}
          rowKey='id'
        />
      </Card>

      <Modal
        title={editingTier ? '编辑层级' : '新建层级'}
        visible={modalVisible}
        onCancel={() => { setModalVisible(false); setEditingTier(null); }}
        footer={null}
        width={500}
      >
        <Form
          getFormApi={(api) => setFormApi(api)}
          onSubmit={handleSubmit}
          layout='vertical'
        >
          <Form.Input
            field='level'
            label='层级编号'
            rules={[{ required: true, message: '请输入层级编号' }]}
            placeholder='如：1, 2, 3, 4'
            type='number'
            initValue={editingTier?.level || ''}
          />
          <Form.Input
            field='name'
            label='层级名称'
            rules={[{ required: true, message: '请输入层级名称' }]}
            placeholder='如：青铜、白银、黄金'
          />
          <Form.Select
            field='color'
            label='颜色'
            initValue='blue'
          >
            {colorOptions.map((c) => (
              <Form.Select.Option key={c} value={c}>
                <Tag color={c}>{c}</Tag>
              </Form.Select.Option>
            ))}
          </Form.Select>
          <Form.Input
            field='min_points'
            label='最低积分要求'
            placeholder='达到该层级所需最低积分'
            type='number'
            initValue={0}
          />
          <Form.Input
            field='description'
            label='描述'
            placeholder='层级描述（可选）'
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 20 }}>
            <Button onClick={() => { setModalVisible(false); setEditingTier(null); }}>取消</Button>
            <Button type='primary' htmlType='submit' loading={submitting}>
              {editingTier ? '保存' : '创建'}
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  );
}
