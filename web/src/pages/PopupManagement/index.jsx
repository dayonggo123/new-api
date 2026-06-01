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
  Switch,
  Select,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconRefresh,
  IconDelete,
  IconEdit,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Title, Text } = Typography;

const typeMap = {
  announcement: { color: 'blue', text: '公告' },
  promo: { color: 'green', text: '活动' },
  update: { color: 'orange', text: '更新' },
};

export default function PopupManagement() {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [editingItem, setEditingItem] = useState(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/popups');
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

  const handleOpenModal = (record = null) => {
    setEditingItem(record);
    setModalVisible(true);
  };

  const handleCloseModal = () => {
    setModalVisible(false);
    setEditingItem(null);
    formApi?.reset();
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      let res;
      const payload = {
        title: values.title,
        content: values.content,
        image_url: values.image_url || '',
        type: values.type || 'announcement',
        enabled: !!values.enabled,
      };

      if (editingItem) {
        res = await API.put('/api/admin/popups', { ...payload, id: editingItem.id });
      } else {
        res = await API.post('/api/admin/popups', payload);
      }

      const { success, message } = res.data;
      if (success) {
        showSuccess(message || (editingItem ? '更新成功' : '创建成功'));
        handleCloseModal();
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
      const res = await API.delete(`/api/admin/popups/${id}`);
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

  // Modal 打开后设置表单值
  useEffect(() => {
    if (!modalVisible || !formApi) return;
    if (editingItem) {
      formApi.setValues({
        title: editingItem.title,
        content: editingItem.content,
        image_url: editingItem.image_url || '',
        type: editingItem.type || 'announcement',
        enabled: editingItem.enabled,
      });
    } else {
      formApi.reset();
      formApi.setValues({
        type: 'announcement',
        enabled: true,
      });
    }
  }, [modalVisible, formApi, editingItem]);

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '内容',
      dataIndex: 'content',
      ellipsis: true,
      render: (text) => (
        <span style={{ maxWidth: 300, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {text}
        </span>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 100,
      render: (type) => {
        const config = typeMap[type] || { color: 'grey', text: type };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 80,
      render: (enabled) => (
        <Tag color={enabled ? 'green' : 'red'}>
          {enabled ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (time) => {
        if (!time) return '-';
        return new Date(time * 1000).toLocaleString('zh-CN');
      },
    },
    {
      title: '操作',
      width: 120,
      render: (_, record) => (
        <Space>
          <Button icon={<IconEdit />} theme='light' onClick={() => handleOpenModal(record)} />
          <Popconfirm
            title='确认删除'
            content={`确定要删除弹窗 "${record.title}" 吗？`}
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
            <Title heading={4} style={{ margin: 0 }}>弹窗管理</Title>
            <Text type='tertiary' size='small'>管理首页 Daily Popup 弹窗内容</Text>
          </div>
        }
        headerExtraContent={
          <Space>
            <Button icon={<IconRefresh />} onClick={loadData} loading={loading}>
              刷新
            </Button>
            <Button type='primary' icon={<IconPlus />} onClick={() => handleOpenModal()}>
              新建弹窗
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          empty={<Empty description='暂无弹窗' />}
          rowKey='id'
        />
      </Card>

      <Modal
        title={editingItem ? '编辑弹窗' : '新建弹窗'}
        visible={modalVisible}
        onCancel={handleCloseModal}
        footer={null}
        width={600}
      >
        <Form
          getFormApi={(api) => setFormApi(api)}
          onSubmit={handleSubmit}
          layout='vertical'
        >
          <Form.Input
            field='title'
            label='标题'
            rules={[{ required: true, message: '请输入弹窗标题' }]}
            placeholder='如：今日公告'
          />
          <Form.TextArea
            field='content'
            label='内容'
            rules={[{ required: true, message: '请输入弹窗内容' }]}
            placeholder='支持多行文本'
            rows={4}
          />
          <Form.Input
            field='image_url'
            label='图片 URL'
            placeholder='可选：弹窗顶部图片地址'
          />
          <Form.Select
            field='type'
            label='类型'
            initValue='announcement'
            optionList={[
              { value: 'announcement', label: '公告' },
              { value: 'promo', label: '活动' },
              { value: 'update', label: '更新' },
            ]}
          />
          <Form.Switch
            field='enabled'
            label='启用'
            initValue={true}
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 20 }}>
            <Button onClick={handleCloseModal}>取消</Button>
            <Button type='primary' htmlType='submit' loading={submitting}>
              {editingItem ? '更新' : '创建'}
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  );
}
