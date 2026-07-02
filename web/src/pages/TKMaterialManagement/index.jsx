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

// 一级分类 + 二级分类
const CATEGORY_GROUPS = [
  {
    label: '场景',
    value: '场景',
    children: ['浴室', '客厅', '厨房', '卧室', '车库', '院子', '街景', '健身房', '车', '机场', '农村', '公园', '超市', '仓库'],
  },
  {
    label: '分析 UGC',
    value: '分析 UGC',
    children: ['男', '女'],
  },
];

// 构建完整的 category 存储值
function buildCategory(groupValue, childValue) {
  if (!childValue) return '';
  if (groupValue === '分析 UGC') {
    return `分析 UGC/${childValue}`;
  }
  return childValue;
}

// 解析完整 category 为分组+子分类
function parseCategory(category) {
  if (!category) return { group: '', child: '' };
  if (category.startsWith('分析 UGC/')) {
    return { group: '分析 UGC', child: category.replace('分析 UGC/', '') };
  }
  return { group: '场景', child: category };
}

// 所有扁平化的 category 值（用于统计/导入默认值）
const ALL_CATEGORIES = CATEGORY_GROUPS.flatMap((g) =>
  g.children.map((child) => buildCategory(g.value, child))
);

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

  const [uploadCategoryGroup, setUploadCategoryGroup] = useState('');

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
        // 如果删除后当前页已经没有数据且不是第一页，回到上一页
        if (data.length === 1 && page > 1) {
          setPage(page - 1);
        } else {
          loadData();
        }
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
    const selectedCategory = buildCategory(values.categoryGroup, values.categoryChild);
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
        categories: values.categories || ALL_CATEGORIES,
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
          onError={(e) => {
            e.target.src = '';
            e.target.alt = t('加载失败');
          }}
        />
      ),
    },
    {
      title: t('分类'),
      dataIndex: 'category',
      width: 140,
      render: (cat) => {
        const { group, child } = parseCategory(cat);
        return (
          <Space>
            {group && <Tag color='light-blue'>{group}</Tag>}
            <Tag color='blue'>{child}</Tag>
          </Space>
        );
      },
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

  // 当前选中的分组和子分类
  const { group: selectedGroup, child: selectedChild } = parseCategory(category);

  const groupOptions = CATEGORY_GROUPS.map((g) => ({ value: g.value, label: g.label }));
  const childOptions = (CATEGORY_GROUPS.find((g) => g.value === selectedGroup)?.children || []).map(
    (child) => ({ value: child, label: child })
  );

  // 上传表单用的分组/子分类选项
  const uploadGroupOptions = CATEGORY_GROUPS.map((g) => ({ value: g.value, label: g.label }));

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
              {CATEGORY_GROUPS.map((group) =>
                group.children.map((child) => {
                  const fullCat = buildCategory(group.value, child);
                  return (
                    <Tag key={fullCat} color='light-blue'>
                      {group.label}/{child}: {stats[fullCat] || 0}
                    </Tag>
                  );
                })
              )}
            </Space>
          </div>
        }
        style={{ marginBottom: 24 }}
      />

      <Card style={{ marginBottom: 24 }}>
        <Space wrap>
          <Select
            placeholder={t('选择一级分类')}
            style={{ width: 140 }}
            value={selectedGroup}
            onChange={(v) => {
              setCategory(buildCategory(v, ''));
              setPage(1);
            }}
            optionList={[{ value: '', label: t('全部分类') }, ...groupOptions]}
          />
          <Select
            placeholder={t('选择二级分类')}
            style={{ width: 140 }}
            value={selectedChild}
            disabled={!selectedGroup}
            onChange={(v) => {
              setCategory(buildCategory(selectedGroup, v));
              setPage(1);
            }}
            optionList={[{ value: '', label: t('全部') }, ...childOptions]}
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
            field='categoryGroup'
            label={t('一级分类')}
            placeholder={t('请选择一级分类')}
            rules={[{ required: true, message: t('请选择一级分类') }]}
            optionList={uploadGroupOptions}
            onChange={(v) => {
              setUploadCategoryGroup(v);
              formApi?.setValue('categoryChild', '');
            }}
            style={{ width: '100%' }}
          />
          <Form.Select
            field='categoryChild'
            label={t('二级分类')}
            placeholder={t('请选择二级分类')}
            rules={[{ required: true, message: t('请选择二级分类') }]}
            optionList={[
              { value: '', label: t('请选择') },
              ...(CATEGORY_GROUPS.find((g) => g.value === uploadCategoryGroup)?.children || []).map(
                (child) => ({ value: child, label: child })
              ),
            ]}
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
              initValue={ALL_CATEGORIES}
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
