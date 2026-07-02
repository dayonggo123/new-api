import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Button,
  Card,
  Input,
  Table,
  Tag,
  Space,
  Modal,
  Popconfirm,
  Typography,
  Empty,
  Pagination,
  Select,
  Upload,
  Form,
  Descriptions,
  Spin,
  Banner,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconRefresh,
  IconDelete,
  IconImport,
  IconCopy,
  IconSearch,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess, copy } from '../../helpers';
import { useTranslation } from 'react-i18next';

const { Title, Text } = Typography;

const CATEGORIES = [
  '浴室', '客厅', '厨房', '卧室', '车库', '院子',
  '街景', '健身房', '车', '机场', '农村', '公园',
  '超市', '仓库',
];

export default function TKMaterialManagement() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [category, setCategory] = useState('');
  const [keyword, setKeyword] = useState('');
  const [stats, setStats] = useState({});
  const [uploadVisible, setUploadVisible] = useState(false);
  const [importVisible, setImportVisible] = useState(false);
  const [importLoading, setImportLoading] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [importFormApi, setImportFormApi] = useState(null);
  const [fileList, setFileList] = useState([]);
  const uploadRef = useRef(null);

  const loadStats = useCallback(async () => {
    try {
      const res = await API.get('/api/admin/tk/materials/stats');
      if (res.data.success) {
        setStats(res.data.data || {});
      }
    } catch (error) {
      console.error('load stats failed', error);
    }
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const params = { page, page_size: pageSize };
      if (category) params.category = category;
      if (keyword) params.keyword = keyword;
      const res = await API.get('/api/admin/tk/materials', { params });
      const { success, data: result } = res.data;
      if (success) {
        setData(result.data || []);
        setTotal(result.total || 0);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setLoading(false);
  }, [page, pageSize, category, keyword]);

  useEffect(() => {
    loadData();
    loadStats();
  }, [loadData, loadStats]);

  const handleDelete = async (id) => {
    try {
      const res = await API.delete(`/api/admin/tk/materials/${id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        loadData();
        loadStats();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
  };

  const handleCopy = async (url) => {
    if (await copy(url)) {
      showSuccess(t('URL 已复制'));
    } else {
      showError(t('复制失败'));
    }
  };

  const handleUpload = async () => {
    if (!formApi) return;
    const values = formApi.getValues();
    const selectedCategory = values.category;
    if (!selectedCategory) {
      showError(t('请选择分类'));
      return;
    }
    if (fileList.length === 0) {
      showError(t('请选择要上传的图片'));
      return;
    }
    const formData = new FormData();
    formData.append('category', selectedCategory);
    fileList.forEach((file) => {
      formData.append('files', file.fileInstance);
    });
    try {
      const res = await API.post('/api/admin/tk/materials', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        params: { permanent: true },
      });
      if (res.data.success) {
        showSuccess(t('上传成功'));
        setUploadVisible(false);
        setFileList([]);
        formApi.reset();
        setPage(1);
        loadData();
        loadStats();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
  };

  const handleImport = async () => {
    if (!importFormApi) return;
    const values = importFormApi.getValues();
    if (!values.database_id) {
      showError(t('请输入 Notion Database ID'));
      return;
    }
    setImportLoading(true);
    try {
      const res = await API.post('/api/admin/tk/materials/import/notion', {
        token: values.token,
        database_id: values.database_id,
        categories: values.categories || CATEGORIES,
      });
      if (res.data.success) {
        const result = res.data.data;
        showSuccess(
          t('导入完成：共 {{total}} 页，新增 {{imported}} 条，重复 {{duplicate}} 条，失败 {{error}} 条', {
            total: result.total_pages,
            imported: result.imported_count,
            duplicate: result.duplicate_count,
            error: result.error_count,
          }),
        );
        setImportVisible(false);
        importFormApi.reset();
        setPage(1);
        loadData();
        loadStats();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error?.message || error);
    }
    setImportLoading(false);
  };

  const columns = [
    {
      title: t('ID'),
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('预览'),
      dataIndex: 'url',
      width: 120,
      render: (url) => (
        <img
          src={url}
          alt='material'
          style={{ width: 80, height: 80, objectFit: 'cover', borderRadius: 8 }}
        />
      ),
    },
    {
      title: t('分类'),
      dataIndex: 'category',
      width: 120,
      render: (cat) => <Tag color='blue'>{cat}</Tag>,
    },
    {
      title: t('来源'),
      dataIndex: 'source',
      width: 100,
      render: (source) => (
        <Tag color={source === 'notion' ? 'purple' : 'green'}>
          {source === 'notion' ? t('Notion') : t('上传')}
        </Tag>
      ),
    },
    {
      title: t('文件名'),
      dataIndex: 'filename',
      ellipsis: true,
    },
    {
      title: t('URL'),
      dataIndex: 'url',
      ellipsis: true,
      render: (url) => (
        <Text
          copyable
          style={{ cursor: 'pointer' }}
          onClick={() => handleCopy(url)}
        >
          {url}
        </Text>
      ),
    },
    {
      title: t('时间'),
      dataIndex: 'created_at',
      width: 180,
      render: (ts) => new Date(ts * 1000).toLocaleString(),
    },
    {
      title: t('操作'),
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            icon={<IconCopy />}
            size='small'
            onClick={() => handleCopy(record.url)}
          >
            {t('复制')}
          </Button>
          <Popconfirm
            title={t('确认删除')}
            content={t('删除后不可恢复，是否继续？')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button icon={<IconDelete />} type='danger' size='small'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const categoryOptions = CATEGORIES.map((cat) => ({ value: cat, label: cat }));

  return (
    <div className='p-6'>
      <Title heading={3} style={{ marginBottom: 24 }}>
        {t('TK 素材库')}
      </Title>

      <Banner
        type='info'
        closeIcon={null}
        description={
          <div>
            <Text strong>{t('分类统计：')}</Text>
            <Space wrap>
              {CATEGORIES.map((cat) => (
                <Tag key={cat} color='light-blue'>
                  {cat}: {stats[cat] || 0}
                </Tag>
              ))}
            </Space>
          </div>
        }
        style={{ marginBottom: 24 }}
      />

      <Card style={{ marginBottom: 24 }}>
        <Space wrap>
          <Select
            placeholder={t('按分类筛选')}
            style={{ width: 160 }}
            value={category}
            onChange={(v) => {
              setCategory(v);
              setPage(1);
            }}
            optionList={[{ value: '', label: t('全部分类') }, ...categoryOptions]}
          />
          <Input
            prefix={<IconSearch />}
            placeholder={t('搜索文件名或分类')}
            value={keyword}
            onChange={(v) => setKeyword(v)}
            onEnterPress={() => {
              setPage(1);
              loadData();
            }}
            style={{ width: 240 }}
          />
          <Button icon={<IconRefresh />} onClick={() => loadData()}>
            {t('刷新')}
          </Button>
          <Button
            icon={<IconPlus />}
            theme='solid'
            onClick={() => setUploadVisible(true)}
          >
            {t('上传素材')}
          </Button>
          <Button
            icon={<IconImport />}
            onClick={() => setImportVisible(true)}
          >
            {t('Notion 导入')}
          </Button>
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={false}
          rowKey='id'
          empty={<Empty description={t('暂无素材')} />}
          scroll={{ x: 1200 }}
        />
        <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end' }}>
          <Pagination
            total={total}
            currentPage={page}
            pageSize={pageSize}
            pageSizeOpts={[10, 20, 50, 100]}
            onPageChange={(p) => setPage(p)}
            onPageSizeChange={(s) => {
              setPageSize(s);
              setPage(1);
            }}
          />
        </div>
      </Card>

      {/* Upload Modal */}
      <Modal
        title={t('上传素材')}
        visible={uploadVisible}
        onCancel={() => {
          setUploadVisible(false);
          setFileList([]);
          formApi?.reset();
        }}
        footer={
          <Space>
            <Button onClick={() => setUploadVisible(false)}>{t('取消')}</Button>
            <Button theme='solid' onClick={handleUpload}>
              {t('上传')}
            </Button>
          </Space>
        }
      >
        <Form getFormApi={setFormApi}>
          <Form.Select
            field='category'
            label={t('分类')}
            placeholder={t('请选择分类')}
            rules={[{ required: true, message: t('请选择分类') }]}
            optionList={categoryOptions}
            style={{ width: '100%' }}
          />
          <Form.Slot label={t('图片')}>
            <Upload
              ref={uploadRef}
              action='#'
              accept='image/*'
              multiple
              fileList={fileList}
              onChange={({ fileList }) => setFileList(fileList)}
              beforeUpload={() => false}
              listType='picture'
            >
              <Button icon={<IconPlus />} theme='light'>
                {t('选择图片')}
              </Button>
            </Upload>
          </Form.Slot>
        </Form>
      </Modal>

      {/* Import Modal */}
      <Modal
        title={t('从 Notion 导入')}
        visible={importVisible}
        onCancel={() => {
          setImportVisible(false);
          importFormApi?.reset();
        }}
        footer={
          <Space>
            <Button onClick={() => setImportVisible(false)}>{t('取消')}</Button>
            <Button
              theme='solid'
              onClick={handleImport}
              loading={importLoading}
            >
              {t('开始导入')}
            </Button>
          </Space>
        }
      >
        <Spin spinning={importLoading}>
          <Form getFormApi={setImportFormApi}>
            <Form.Input
              field='database_id'
              label={t('Notion Database ID')}
              placeholder={t('请输入 Database ID')}
              rules={[{ required: true, message: t('请输入 Database ID') }]}
            />
            <Form.Input
              field='token'
              label={t('Notion Integration Token')}
              placeholder={t('留空则读取环境变量 NOTION_INTEGRATION_TOKEN')}
            />
            <Form.TagInput
              field='categories'
              label={t('导入分类（列名）')}
              placeholder={t('输入列名后回车')}
              initValue={CATEGORIES}
              style={{ width: '100%' }}
            />
          </Form>
          <Descriptions
            data={[
              {
                key: '提示',
                value: t('Notion 表格中列名即分类名，单元格内支持 files/url/rich_text 类型的图片地址'),
              },
            ]}
            row
          />
        </Spin>
      </Modal>
    </div>
  );
}
