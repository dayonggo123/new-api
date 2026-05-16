import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  SideSheet,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
  Upload,
  Row,
  Col,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconDelete,
  IconTickCircle,
  IconUpload,
  IconEdit,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Text, Title } = Typography;

const PLATFORMS = [
  { value: 'windows', label: 'Windows' },
  { value: 'darwin', label: 'macOS' },
  { value: 'linux', label: 'Linux' },
];

const ARCHS = [
  { value: 'x86_64', label: 'x86_64' },
  { value: 'aarch64', label: 'aarch64 (ARM64)' },
];

const PLATFORM_COLORS = {
  windows: 'blue',
  darwin: 'purple',
  linux: 'orange',
};

export default function AppReleaseManagement() {
  const { t } = useTranslation();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [total, setTotal] = useState(0);
  const [showUpload, setShowUpload] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadFile, setUploadFile] = useState(null);
  const [showEdit, setShowEdit] = useState(false);
  const [editingRecord, setEditingRecord] = useState(null);
  const [savingEdit, setSavingEdit] = useState(false);

  const loadData = useCallback(
    async (page = activePage, size = pageSize) => {
      setLoading(true);
      try {
        const res = await API.get(`/api/admin/releases?p=${page}&page_size=${size}`);
        const { success, data, message } = res.data;
        if (success) {
          setItems(data.items || []);
          setActivePage(data.page <= 0 ? 1 : data.page);
          setTotal(data.total || 0);
        } else {
          showError(message);
        }
      } catch (error) {
        showError(error?.message || error);
      }
      setLoading(false);
    },
    [activePage, pageSize]
  );

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleDelete = async (id) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('删除后无法恢复，是否继续？'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/admin/releases/${id}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            loadData();
          } else {
            showError(res.data.message);
          }
        } catch (error) {
          showError(error?.message || error);
        }
      },
    });
  };

  const handleMarkLatest = async (id) => {
    try {
      const res = await API.put(`/api/admin/releases/${id}/latest`);
      if (res.data.success) {
        showSuccess(t('已标记为最新版'));
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
  };

  const handleUpload = async (values) => {
    if (!uploadFile) {
      showError(t('请选择安装包文件'));
      return;
    }
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append('version', values.version);
      formData.append('tag', values.tag);
      formData.append('platform', values.platform);
      formData.append('arch', values.arch);
      formData.append('file', uploadFile);
      if (values.release_notes) {
        formData.append('release_notes', values.release_notes);
      }
      if (values.signature) {
        formData.append('signature', values.signature);
      }
      formData.append('is_force', values.is_force ? 'true' : 'false');

      const res = await API.post('/api/admin/releases', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (res.data.success) {
        showSuccess(t('上传成功'));
        setShowUpload(false);
        setUploadFile(null);
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setUploading(false);
  };

  const handleEdit = async (values) => {
    if (!editingRecord) return;
    setSavingEdit(true);
    try {
      const payload = {
        version: values.version,
        tag: values.tag,
        release_notes: values.release_notes || '',
        signature: values.signature || '',
        is_force: values.is_force,
      };
      const res = await API.put(`/api/admin/releases/${editingRecord.id}`, payload);
      if (res.data.success) {
        showSuccess(t('保存成功'));
        setShowEdit(false);
        setEditingRecord(null);
        loadData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setSavingEdit(false);
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('版本号'),
      dataIndex: 'version',
      render: (v, record) => (
        <Space>
          <Text strong>{v}</Text>
          {record.is_latest && <Tag color='green'>{t('最新')}</Tag>}
          {record.is_force && <Tag color='red'>{t('强制')}</Tag>}
        </Space>
      ),
    },
    {
      title: t('平台'),
      dataIndex: 'platform',
      width: 100,
      render: (v) => <Tag color={PLATFORM_COLORS[v] || 'default'}>{v}</Tag>,
    },
    {
      title: t('架构'),
      dataIndex: 'arch',
      width: 100,
    },
    {
      title: t('文件名'),
      dataIndex: 'file_name',
      render: (v) => <Text size='small'>{v}</Text>,
    },
    {
      title: t('大小'),
      dataIndex: 'file_size',
      width: 100,
      render: (v) => <Text size='small'>{(v / 1024 / 1024).toFixed(2)} MB</Text>,
    },
    {
      title: t('下载链接'),
      dataIndex: 'download_url',
      render: (v) => (
        <Text size='small' link onClick={() => window.open(v, '_blank')}>
          {v}
        </Text>
      ),
    },
    {
      title: t('操作'),
      width: 220,
      render: (_, record) => (
        <Space>
          {!record.is_latest && (
            <Button
              type='tertiary'
              size='small'
              icon={<IconTickCircle />}
              onClick={() => handleMarkLatest(record.id)}
            >
              {t('设为最新')}
            </Button>
          )}
          <Button
            type='tertiary'
            size='small'
            icon={<IconEdit />}
            onClick={() => {
              setEditingRecord(record);
              setShowEdit(true);
            }}
          >
            {t('编辑')}
          </Button>
          <Button
            type='danger'
            size='small'
            icon={<IconDelete />}
            onClick={() => handleDelete(record.id)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card
        title={
          <div className='flex items-center justify-between'>
            <Title heading={4} style={{ margin: 0 }}>{t('安装包管理')}</Title>
            <Button theme='solid' icon={<IconPlus />} onClick={() => setShowUpload(true)}>
              {t('上传新版本')}
            </Button>
          </div>
        }
        className='!rounded-2xl shadow-sm'
      >
        <Table
          loading={loading}
          columns={columns}
          dataSource={items}
          pagination={{
            currentPage: activePage,
            pageSize: pageSize,
            total: total,
            onPageChange: (page) => {
              setActivePage(page);
              loadData(page, pageSize);
            },
            onPageSizeChange: (size) => {
              setPageSize(size);
              setActivePage(1);
              loadData(1, size);
            },
          }}
        />
      </Card>

      {/* Upload SideSheet */}
      <SideSheet
        title={t('上传新版本')}
        visible={showUpload}
        onCancel={() => { setShowUpload(false); setUploadFile(null); }}
        width={560}
        footer={
          <div className='flex justify-end'>
            <Space>
              <Button theme='light' onClick={() => { setShowUpload(false); setUploadFile(null); }}>
                {t('取消')}
              </Button>
              <Button
                theme='solid'
                loading={uploading}
                onClick={() => {
                  const form = document.getElementById('release-upload-form');
                  if (form) form.dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }));
                }}
              >
                {t('上传')}
              </Button>
            </Space>
          </div>
        }
      >
        <Form
          id='release-upload-form'
          onSubmit={handleUpload}
          initValues={{ platform: 'windows', arch: 'x86_64', is_force: false }}
        >
          {({ formApi }) => (
            <div className='p-2'>
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Input
                    field='version'
                    label={t('版本号')}
                    placeholder='1.2.3'
                    rules={[{ required: true, message: t('请输入版本号') }]}
                  />
                </Col>
                <Col span={12}>
                  <Form.Input
                    field='tag'
                    label={t('Tag')}
                    placeholder='v1.2.3'
                    rules={[{ required: true, message: t('请输入 Tag') }]}
                  />
                </Col>
              </Row>
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Select
                    field='platform'
                    label={t('平台')}
                    rules={[{ required: true }]}
                  >
                    {PLATFORMS.map((p) => (
                      <Form.Select.Option key={p.value} value={p.value}>
                        {p.label}
                      </Form.Select.Option>
                    ))}
                  </Form.Select>
                </Col>
                <Col span={12}>
                  <Form.Select
                    field='arch'
                    label={t('架构')}
                    rules={[{ required: true }]}
                  >
                    {ARCHS.map((a) => (
                      <Form.Select.Option key={a.value} value={a.value}>
                        {a.label}
                      </Form.Select.Option>
                    ))}
                  </Form.Select>
                </Col>
              </Row>
              <Form.TextArea
                field='release_notes'
                label={t('更新日志')}
                placeholder={t('支持 Markdown')}
                rows={4}
              />
              <Form.TextArea
                field='signature'
                label={t('签名内容')}
                placeholder={t('粘贴 .sig 文件内容')}
                rows={4}
              />
              <Form.Switch
                field='is_force'
                label={t('强制升级')}
              />
              <div style={{ marginTop: 16 }}>
                <Text type='tertiary' size='small' style={{ marginBottom: 8, display: 'block' }}>
                  {t('安装包文件')}
                </Text>
                <Upload
                  accept='.exe,.dmg,.zip,.tar.gz,.deb,.rpm,.AppImage'
                  limit={1}
                  customRequest={({ file, onSuccess, onError }) => {
                    if (file?.fileInstance) {
                      setUploadFile(file.fileInstance);
                      onSuccess();
                    } else {
                      onError();
                    }
                  }}
                  onRemove={() => {
                    setUploadFile(null);
                  }}
                  fileList={uploadFile ? [{ name: uploadFile.name, uid: uploadFile.name, status: 'success', fileInstance: uploadFile }] : []}
                >
                  <Button icon={<IconUpload />} type='tertiary'>
                    {t('选择文件')}
                  </Button>
                </Upload>
              </div>
            </div>
          )}
        </Form>
      </SideSheet>

      {/* Edit SideSheet */}
      <SideSheet
        title={t('编辑版本')}
        visible={showEdit}
        onCancel={() => { setShowEdit(false); setEditingRecord(null); }}
        width={560}
        footer={
          <div className='flex justify-end'>
            <Space>
              <Button theme='light' onClick={() => { setShowEdit(false); setEditingRecord(null); }}>
                {t('取消')}
              </Button>
              <Button
                theme='solid'
                loading={savingEdit}
                onClick={() => {
                  const form = document.getElementById('release-edit-form');
                  if (form) form.dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }));
                }}
              >
                {t('保存')}
              </Button>
            </Space>
          </div>
        }
      >
        {editingRecord && (
          <Form
            id='release-edit-form'
            onSubmit={handleEdit}
            initValues={{
              version: editingRecord.version,
              tag: editingRecord.tag,
              release_notes: editingRecord.release_notes || '',
              signature: editingRecord.signature || '',
              is_force: editingRecord.is_force,
            }}
          >
            {({ formApi }) => (
              <div className='p-2'>
                <Row gutter={12}>
                  <Col span={12}>
                    <Form.Input
                      field='version'
                      label={t('版本号')}
                      placeholder='1.2.3'
                      rules={[{ required: true, message: t('请输入版本号') }]}
                    />
                  </Col>
                  <Col span={12}>
                    <Form.Input
                      field='tag'
                      label={t('Tag')}
                      placeholder='v1.2.3'
                      rules={[{ required: true, message: t('请输入 Tag') }]}
                    />
                  </Col>
                </Row>
                <Row gutter={12}>
                  <Col span={12}>
                    <div style={{ marginBottom: 12 }}>
                      <Text type='secondary' size='small' style={{ display: 'block', marginBottom: 4 }}>
                        {t('平台')}
                      </Text>
                      <Tag color={PLATFORM_COLORS[editingRecord.platform] || 'default'}>
                        {editingRecord.platform}
                      </Tag>
                    </div>
                  </Col>
                  <Col span={12}>
                    <div style={{ marginBottom: 12 }}>
                      <Text type='secondary' size='small' style={{ display: 'block', marginBottom: 4 }}>
                        {t('架构')}
                      </Text>
                      <Text>{editingRecord.arch}</Text>
                    </div>
                  </Col>
                </Row>
                <Form.TextArea
                  field='release_notes'
                  label={t('更新日志')}
                  placeholder={t('支持 Markdown')}
                  rows={4}
                />
                <Form.TextArea
                  field='signature'
                  label={t('签名内容')}
                  placeholder={t('粘贴 .sig 文件内容')}
                  rows={4}
                />
                <Form.Switch
                  field='is_force'
                  label={t('强制升级')}
                />
              </div>
            )}
          </Form>
        )}
      </SideSheet>
    </div>
  );
}
