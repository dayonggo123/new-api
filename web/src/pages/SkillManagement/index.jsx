import React, { useState, useEffect, useCallback } from 'react';
import {
  Button,
  Card,
  Table,
  Modal,
  Form,
  Input,
  TextArea,
  TagInput,
  Switch,
  Space,
  Typography,
  Empty,
} from '@douyinfe/semi-ui';
import { IconPlus, IconRefresh } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const NODE_TYPES = [
  'uploadImageNode',
  'imageEditNode',
  'videoGenNode',
  'exportImageNode',
  'llmAgentNode',
  'textAnnotationNode',
  'storyboardGenNode',
  'storyboardSplitNode',
  'groupNode',
  'productEditNode',
];

export default function SkillManagement() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [formApi, setFormApi] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/skills/all');
      const { success, message, data: result } = res.data;
      if (success) {
        setData(result || []);
      } else {
        showError(message);
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleAdd = () => {
    setEditingItem(null);
    setModalVisible(true);
    if (formApi) formApi.reset();
  };

  const handleEdit = (record) => {
    setEditingItem(record);
    setModalVisible(true);
  };

  useEffect(() => {
    if (modalVisible && editingItem && formApi) {
      formApi.setValues({
        id: editingItem.id,
        name: editingItem.name,
        nameEn: editingItem.nameEn || '',
        icon: editingItem.icon,
        cost: editingItem.cost,
        supportedNodeTypes: editingItem.supportedNodeTypes || [],
        description: editingItem.description || '',
        executionType: editingItem.execution?.type || 'llm',
        systemPromptTemplate: editingItem.execution?.systemPromptTemplate || '',
        userPromptTemplate: editingItem.execution?.userPromptTemplate || '',
        overrideLocal: editingItem.overrideLocal || false,
        status: editingItem.status !== 2 ? 1 : 2,
      });
    }
    if (modalVisible && !editingItem && formApi) {
      formApi.reset();
      formApi.setValues({
        executionType: 'llm',
        cost: 0,
        status: 1,
        overrideLocal: false,
        supportedNodeTypes: [],
      });
    }
  }, [modalVisible, editingItem, formApi]);

  const handleDelete = (record) => {
    Modal.confirm({
      title: t('确认删除'),
      content: `${t('确定要删除 Skill')}「${record.name}」？`,
      onOk: async () => {
        try {
          const res = await API.delete(`/api/skills/${record.id}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            loadData();
          } else {
            showError(res.data.message);
          }
        } catch (err) {
          showError(err.message);
        }
      },
    });
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const payload = {
        id: values.id,
        name: values.name,
        nameEn: values.nameEn || '',
        icon: values.icon,
        cost: values.cost || 0,
        supportedNodeTypes: values.supportedNodeTypes || [],
        description: values.description || '',
        execution: {
          type: values.executionType || 'llm',
          systemPromptTemplate: values.systemPromptTemplate || '',
          userPromptTemplate: values.userPromptTemplate || '',
        },
        overrideLocal: values.overrideLocal || false,
        status: values.status || 1,
      };

      let res;
      if (editingItem) {
        res = await API.put(`/api/skills/${editingItem.id}`, payload);
      } else {
        res = await API.post('/api/skills', payload);
      }

      if (res.data.success) {
        showSuccess(editingItem ? t('更新成功') : t('创建成功'));
        setModalVisible(false);
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (err) {
      showError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: t('名称'), dataIndex: 'name', key: 'name' },
    { title: t('图标'), dataIndex: 'icon', key: 'icon' },
    { title: t('消耗'), dataIndex: 'cost', key: 'cost' },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
      render: (v, record) => (
        <Text type={record.status !== 2 ? 'success' : 'danger'}>
          {record.status !== 2 ? t('启用') : t('禁用')}
        </Text>
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button type='tertiary' size='small' onClick={() => handleEdit(record)}>
            {t('编辑')}
          </Button>
          <Button type='danger' size='small' onClick={() => handleDelete(record)}>
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card
        title={t('Skill 管理')}
        headerExtraContent={
          <Space>
            <Button icon={<IconRefresh />} onClick={loadData}>
              {t('刷新')}
            </Button>
            <Button type='primary' icon={<IconPlus />} onClick={handleAdd}>
              {t('新增 Skill')}
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          empty={<Empty description={t('暂无数据')} />}
        />
      </Card>

      <Modal
        title={editingItem ? t('编辑 Skill') : t('新增 Skill')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => formApi && formApi.submitForm()}
        confirmLoading={submitting}
        centered
        maskClosable={false}
        width={720}
        bodyStyle={{ maxHeight: '70vh', overflow: 'auto' }}
      >
        <Form
          getFormApi={(api) => setFormApi(api)}
          onSubmit={handleSubmit}
          layout='vertical'
        >
          <Form.Input
            field='id'
            label='ID'
            placeholder={t('唯一标识，如 prompt-translate')}
            rules={[{ required: true, message: t('ID 不能为空') }]}
            disabled={!!editingItem}
          />
          <Form.Input
            field='name'
            label={t('名称')}
            placeholder={t('中文显示名')}
            rules={[{ required: true, message: t('名称不能为空') }]}
          />
          <Form.Input
            field='nameEn'
            label={t('英文名称')}
            placeholder={t('英文显示名')}
          />
          <Form.Input
            field='icon'
            label={t('图标')}
            placeholder={t('Lucide 图标名，如 languages')}
            rules={[{ required: true, message: t('图标不能为空') }]}
          />
          <Form.InputNumber
            field='cost'
            label={t('消耗积分')}
            placeholder={t('0 = 免费')}
            min={0}
          />
          <Form.TagInput
            field='supportedNodeTypes'
            label={t('支持节点类型')}
            placeholder={t('选择支持的节点类型')}
          />
          <Form.TextArea
            field='description'
            label={t('描述')}
            placeholder={t('Skill 描述')}
            rows={2}
          />
          <Form.Section text={t('执行配置')}>
            <Form.Input
              field='executionType'
              label={t('执行类型')}
              initValue='llm'
              disabled
            />
            <Form.TextArea
              field='systemPromptTemplate'
              label={t('System Prompt 模板')}
              placeholder={t('留空则不设置')}
              rows={4}
            />
            <Form.TextArea
              field='userPromptTemplate'
              label={t('User Prompt 模板')}
              placeholder={t('留空则不设置')}
              rows={4}
            />
          </Form.Section>
          <Form.Switch
            field='overrideLocal'
            label={t('覆盖本地同名 Skill')}
          />
          <Form.Select
            field='status'
            label={t('状态')}
            initValue={1}
          >
            <Form.Select.Option value={1}>{t('启用')}</Form.Select.Option>
            <Form.Select.Option value={2}>{t('禁用')}</Form.Select.Option>
          </Form.Select>
        </Form>
      </Modal>
    </div>
  );
}
