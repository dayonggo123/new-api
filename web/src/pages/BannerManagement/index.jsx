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
  InputNumber,
  TextArea,
  Tabs,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconRefresh,
  IconDelete,
  IconEdit,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

const actionTypeOptions = [
  { value: 'open_url', label: '外部链接 (open_url)' },
  { value: 'open_price_table', label: '价格表 (open_price_table)' },
  { value: 'open_invite', label: '邀请页 (open_invite)' },
  { value: 'open_settings', label: '设置面板 (open_settings)' },
  { value: 'noop', label: '纯展示 (noop)' },
];

const supportedLangs = [
  { code: 'zh', label: '中文' },
  { code: 'en', label: 'English' },
  { code: 'ja', label: '日本語' },
  { code: 'fr', label: 'Français' },
  { code: 'ru', label: 'Русский' },
  { code: 'vi', label: 'Tiếng Việt' },
];

export default function BannerManagement() {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [editingItem, setEditingItem] = useState(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/marketing/banners');
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

  const buildContentFromForm = (values) => {
    const content = {};
    for (const lang of supportedLangs) {
      const text = values[`content_${lang.code}_text`];
      if (text && text.trim()) {
        content[lang.code] = {
          text: text.trim(),
          cta: (values[`content_${lang.code}_cta`] || '').trim(),
          action_type: values[`content_${lang.code}_action_type`] || 'noop',
          action_payload: (values[`content_${lang.code}_action_payload`] || '').trim(),
        };
      }
    }
    return content;
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const content = buildContentFromForm(values);
      if (Object.keys(content).length === 0) {
        showError('请至少填写一种语言的内容');
        setSubmitting(false);
        return;
      }

      const payload = {
        priority: values.priority || 0,
        enabled: !!values.enabled,
        start_at: values.start_at || 0,
        end_at: values.end_at || 0,
        max_dismiss_hours: values.max_dismiss_hours || 24,
        content,
      };

      let res;
      if (editingItem) {
        res = await API.put('/api/admin/marketing/banners', { ...payload, id: editingItem.id });
      } else {
        res = await API.post('/api/admin/marketing/banners', payload);
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
      const res = await API.delete(`/api/admin/marketing/banners/${id}`);
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
      const values = {
        priority: editingItem.priority || 0,
        enabled: editingItem.enabled,
        start_at: editingItem.start_at || 0,
        end_at: editingItem.end_at || 0,
        max_dismiss_hours: editingItem.max_dismiss_hours || 24,
      };
      // 解析 content JSON
      let contentObj = {};
      try {
        contentObj = typeof editingItem.content === 'string'
          ? JSON.parse(editingItem.content)
          : (editingItem.content || {});
      } catch (e) {
        contentObj = {};
      }
      for (const lang of supportedLangs) {
        const c = contentObj[lang.code] || {};
        values[`content_${lang.code}_text`] = c.text || '';
        values[`content_${lang.code}_cta`] = c.cta || '';
        values[`content_${lang.code}_action_type`] = c.action_type || 'noop';
        values[`content_${lang.code}_action_payload`] = c.action_payload || '';
      }
      formApi.setValues(values);
    } else {
      formApi.reset();
      formApi.setValues({
        priority: 0,
        enabled: true,
        start_at: 0,
        end_at: 0,
        max_dismiss_hours: 24,
        content_zh_action_type: 'noop',
        content_en_action_type: 'noop',
      });
    }
  }, [modalVisible, formApi, editingItem]);

  const formatTime = (ts) => {
    if (!ts || ts <= 0) return '-';
    return new Date(ts * 1000).toLocaleString('zh-CN');
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      width: 80,
      render: (p) => <Tag color={p >= 10 ? 'red' : p >= 5 ? 'orange' : 'blue'}>{p}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 80,
      render: (enabled) => (
        <Tag color={enabled ? 'green' : 'red'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '生效时间',
      dataIndex: 'start_at',
      width: 160,
      render: (ts, record) => (
        <span>{formatTime(ts)} ~ {formatTime(record.end_at)}</span>
      ),
    },
    {
      title: '关闭冷却',
      dataIndex: 'max_dismiss_hours',
      width: 100,
      render: (h) => `${h} 小时`,
    },
    {
      title: '内容摘要',
      dataIndex: 'content',
      ellipsis: true,
      render: (content) => {
        let obj = {};
        try {
          obj = typeof content === 'string' ? JSON.parse(content) : (content || {});
        } catch (e) {}
        const firstKey = Object.keys(obj)[0];
        const text = firstKey ? obj[firstKey]?.text : '';
        return (
          <span style={{ maxWidth: 300, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {text || '(无内容)'}
          </span>
        );
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
            content={`确定要删除 Banner ID ${record.id} 吗？`}
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
            <Title heading={4} style={{ margin: 0 }}>运营 Banner 管理</Title>
            <Text type='tertiary' size='small'>管理 App 顶部全局横幅展示内容</Text>
          </div>
        }
        headerExtraContent={
          <Space>
            <Button icon={<IconRefresh />} onClick={loadData} loading={loading}>
              刷新
            </Button>
            <Button type='primary' icon={<IconPlus />} onClick={() => handleOpenModal()}>
              新建 Banner
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          empty={<Empty description='暂无 Banner' />}
          rowKey='id'
        />
      </Card>

      <Modal
        title={editingItem ? '编辑 Banner' : '新建 Banner'}
        visible={modalVisible}
        onCancel={handleCloseModal}
        footer={null}
        width={720}
      >
        <Form
          getFormApi={(api) => setFormApi(api)}
          onSubmit={handleSubmit}
          layout='vertical'
        >
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <Form.InputNumber
              field='priority'
              label='优先级'
              min={0}
              step={1}
              extraText='数字越大越优先展示'
            />
            <Form.InputNumber
              field='max_dismiss_hours'
              label='关闭冷却（小时）'
              min={0}
              step={1}
              extraText='用户关闭后多少小时不再展示'
            />
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <Form.InputNumber
              field='start_at'
              label='生效开始时间（秒级时间戳）'
              min={0}
              step={1}
              extraText='0 表示立即生效'
            />
            <Form.InputNumber
              field='end_at'
              label='生效结束时间（秒级时间戳）'
              min={0}
              step={1}
              extraText='0 表示永久生效'
            />
          </div>
          <Form.Switch
            field='enabled'
            label='启用'
            initValue={true}
          />

          <div style={{ marginTop: 16, marginBottom: 8 }}>
            <Text strong>多语言内容配置</Text>
            <Text type='tertiary' size='small' style={{ display: 'block', marginTop: 4 }}>
              请至少填写一种语言的内容。text 为必填，cta 为按钮文案，action_type 为点击行为，action_payload 为附加参数。
            </Text>
          </div>

          <Tabs type='line'>
            {supportedLangs.map((lang) => (
              <TabPane tab={lang.label} itemKey={lang.code} key={lang.code}>
                <Form.Input
                  field={`content_${lang.code}_text`}
                  label={`主文案 (${lang.code})`}
                  placeholder='如：限时优惠，升级 VIP 享 8 折！'
                />
                <Form.Input
                  field={`content_${lang.code}_cta`}
                  label={`按钮文案 (${lang.code})`}
                  placeholder='如：立即查看'
                />
                <Form.Select
                  field={`content_${lang.code}_action_type`}
                  label={`点击行为 (${lang.code})`}
                  initValue='noop'
                  optionList={actionTypeOptions}
                  style={{ width: '100%' }}
                />
                <Form.Input
                  field={`content_${lang.code}_action_payload`}
                  label={`附加参数 (${lang.code})`}
                  placeholder='open_url 时填链接；open_settings 时填 category 值'
                />
              </TabPane>
            ))}
          </Tabs>

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
