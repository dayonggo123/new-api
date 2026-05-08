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
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Title } = Typography;

const colorOptions = [
  'blue', 'green', 'orange', 'red', 'purple', 'pink', 'cyan', 'yellow',
];

export default function TagManagement() {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formApi, setFormApi] = useState(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/tags');
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
      const res = await API.post('/api/admin/tags', values);
      const { success, message } = res.data;
      if (success) {
        showSuccess(message || '创建成功');
        setModalVisible(false);
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
      const res = await API.delete(`/api/admin/tags/${id}`);
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

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '标签名',
      dataIndex: 'name',
      render: (name, record) => (
        <Tag color={record.color || 'blue'}>{name}</Tag>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      width: 100,
      render: (cat) => (
        <Tag color={cat === 'auto' ? 'orange' : 'blue'}>
          {cat === 'auto' ? '自动' : '手动'}
        </Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'created_time',
      width: 160,
      render: (time) => {
        if (!time) return '-';
        return new Date(time * 1000).toLocaleString('zh-CN');
      },
    },
    {
      title: '操作',
      width: 100,
      render: (_, record) => (
        <Popconfirm
          title='确认删除'
          content={`确定要删除标签 "${record.name}" 吗？关联用户的标签也会被移除。`}
          onConfirm={() => handleDelete(record.id)}
        >
          <Button icon={<IconDelete />} theme='light' type='danger' />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Title heading={4} style={{ margin: 0 }}>用户标签管理</Title>
          </div>
        }
        headerExtraContent={
          <Space>
            <Button icon={<IconRefresh />} onClick={loadData} loading={loading}>
              刷新
            </Button>
            <Button type='primary' icon={<IconPlus />} onClick={() => setModalVisible(true)}>
              新建标签
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          empty={<Empty description='暂无标签' />}
          rowKey='id'
        />
      </Card>

      <Modal
        title='新建标签'
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={500}
      >
        <Form
          getFormApi={(api) => setFormApi(api)}
          onSubmit={handleSubmit}
          layout='vertical'
        >
          <Form.Input
            field='name'
            label='标签名称'
            rules={[{ required: true, message: '请输入标签名称' }]}
            placeholder='如：VIP、新用户、活跃用户'
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
            field='description'
            label='描述'
            placeholder='标签描述（可选）'
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 20 }}>
            <Button onClick={() => setModalVisible(false)}>取消</Button>
            <Button type='primary' htmlType='submit' loading={submitting}>
              创建
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  );
}
