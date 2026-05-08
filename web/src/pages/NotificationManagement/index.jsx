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
  Pagination,
  Select,
  Typography,
  Empty,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconSearch,
  IconRefresh,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Title } = Typography;
const { TextArea } = Input;

const typeMap = {
  system: { color: 'blue', text: '系统' },
  promotion: { color: 'green', text: '活动' },
  announcement: { color: 'orange', text: '公告' },
  task_status: { color: 'purple', text: '任务' },
};

const targetTypeMap = {
  all: '全员广播',
  users: '指定用户',
  group: '指定分组',
};

export default function NotificationManagement() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [modalVisible, setModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [formApi, setFormApi] = useState(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/notifications', {
        params: {
          page,
          page_size: pageSize,
        },
      });
      const { success, data: result } = res.data;
      if (success) {
        setData(result.items || []);
        setTotal(result.total || 0);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  }, [page, pageSize]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const res = await API.post('/api/admin/notifications', values);
      const { success, message } = res.data;
      if (success) {
        showSuccess(message || '发送成功');
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
      title: '目标',
      dataIndex: 'user_id',
      width: 100,
      render: (userId) => (
        <Tag color={userId === 0 ? 'red' : 'blue'}>
          {userId === 0 ? '全员广播' : `用户 ${userId}`}
        </Tag>
      ),
    },
    {
      title: '已读',
      dataIndex: 'is_read',
      width: 80,
      render: (isRead) => (
        <Tag color={isRead ? 'green' : 'orange'}>
          {isRead ? '已读' : '未读'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_time',
      width: 160,
      render: (time) => {
        if (!time) return '-';
        const date = new Date(time * 1000);
        return date.toLocaleString('zh-CN');
      },
    },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Title heading={4} style={{ margin: 0 }}>消息管理</Title>
          </div>
        }
        headerExtraContent={
          <Space>
            <Button
              icon={<IconRefresh />}
              onClick={loadData}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type='primary'
              icon={<IconPlus />}
              onClick={() => setModalVisible(true)}
            >
              发布消息
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          empty={<Empty description='暂无消息' />}
          rowKey='id'
        />
        <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end' }}>
          <Pagination
            total={total}
            currentPage={page}
            pageSize={pageSize}
            onPageChange={(p) => setPage(p)}
            onPageSizeChange={(s) => { setPageSize(s); setPage(1); }}
          />
        </div>
      </Card>

      <Modal
        title='发布消息'
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
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
            rules={[{ required: true, message: '请输入标题' }]}
            placeholder='请输入消息标题'
          />
          <Form.TextArea
            field='content'
            label='内容'
            rules={[{ required: true, message: '请输入内容' }]}
            placeholder='请输入消息内容'
            rows={4}
          />
          <Form.Select
            field='type'
            label='类型'
            rules={[{ required: true, message: '请选择类型' }]}
            placeholder='请选择消息类型'
            initValue='system'
          >
            <Form.Select.Option value='system'>系统</Form.Select.Option>
            <Form.Select.Option value='promotion'>活动</Form.Select.Option>
            <Form.Select.Option value='announcement'>公告</Form.Select.Option>
            <Form.Select.Option value='task_status'>任务</Form.Select.Option>
          </Form.Select>
          <Form.Select
            field='target_type'
            label='发送目标'
            rules={[{ required: true, message: '请选择发送目标' }]}
            placeholder='请选择发送目标'
            initValue='all'
          >
            <Form.Select.Option value='all'>全员广播</Form.Select.Option>
            <Form.Select.Option value='users'>指定用户</Form.Select.Option>
            <Form.Select.Option value='group'>指定分组</Form.Select.Option>
          </Form.Select>
          <Form.Input
            field='action_url'
            label='跳转链接'
            placeholder='可选：点击消息后的跳转链接'
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 20 }}>
            <Button onClick={() => setModalVisible(false)}>取消</Button>
            <Button type='primary' htmlType='submit' loading={submitting}>
              发布
            </Button>
          </div>
        </Form>
      </Modal>
    </div>
  );
}
