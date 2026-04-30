import { useState, useEffect } from 'react';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Table,
  Tag,
  Modal,
  SideSheet,
  Form,
  Input,
  InputNumber,
} from '@douyinfe/semi-ui';
import { Edit, Trash2, Plus } from 'lucide-react';
import { ITEMS_PER_PAGE } from '../../../constants';
import {
  PROMPT_CATEGORY_STATUS,
  PROMPT_CATEGORY_STATUS_MAP,
} from '../../../constants/prompt.constants';
import { CardPro, CardTable } from '../../other';

export default function PromptCategoriesTable() {
  const { t } = useTranslation();
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [total, setTotal] = useState(0);
  const [showEdit, setShowEdit] = useState(false);
  const [editingCategory, setEditingCategory] = useState(null);
  const [formApi, setFormApi] = useState(null);

  const loadCategories = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/prompt-category/?p=${page}&page_size=${size}`,
      );
      if (res.data.success) {
        setCategories(res.data.data.items || []);
        setActivePage(res.data.data.page <= 0 ? 1 : res.data.data.page);
        setTotal(res.data.data.total);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  const handleSave = async () => {
    const values = formApi.getValues();
    if (!values.name) {
      showError(t('分类名称不能为空'));
      return;
    }

    try {
      const payload = {
        ...values,
        status: values.status ? 1 : 2,
      };
      let res;
      if (editingCategory) {
        res = await API.put('/api/prompt-category/', {
          ...payload,
          id: editingCategory.id,
        });
      } else {
        res = await API.post('/api/prompt-category/', payload);
      }
      if (res.data.success) {
        showSuccess(t('保存成功'));
        setShowEdit(false);
        loadCategories(activePage);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleDelete = (id) => {
    Modal.confirm({
      title: t('确认删除'),
      content: t('删除分类后，关联的提示词将变为未分类，是否继续？'),
      onOk: async () => {
        try {
          const res = await API.delete(`/api/prompt-category/${id}`);
          if (res.data.success) {
            showSuccess(t('删除成功'));
            loadCategories(activePage);
          } else {
            showError(res.data.message);
          }
        } catch (error) {
          showError(error.message);
        }
      },
    });
  };

  const openEdit = (category = null) => {
    setEditingCategory(category);
    setShowEdit(true);
  };

  useEffect(() => {
    loadCategories(1, pageSize);
  }, [pageSize]);

  const columns = [
    { title: t('ID'), dataIndex: 'id', width: 60 },
    { title: t('名称'), dataIndex: 'name' },
    { title: t('描述'), dataIndex: 'description' },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (v) => (
        <Tag color={PROMPT_CATEGORY_STATUS_MAP[v]?.color || 'grey'}>
          {PROMPT_CATEGORY_STATUS_MAP[v]?.text || v}
        </Tag>
      ),
    },
    { title: t('排序'), dataIndex: 'sort_order', width: 80 },
    {
      title: t('操作'),
      render: (_, record) => (
        <div className='flex gap-2'>
          <Button
            theme='light'
            type='tertiary'
            icon={<Edit size={14} />}
            onClick={() => openEdit(record)}
          />
          <Button
            theme='light'
            type='danger'
            icon={<Trash2 size={14} />}
            onClick={() => handleDelete(record.id)}
          />
        </div>
      ),
    },
  ];

  const editTitle = editingCategory ? t('编辑分类') : t('新增分类');
  const initValues = editingCategory
    ? { ...editingCategory, status: editingCategory.status === 1 }
    : { status: true, sort_order: 0 };

  return (
    <CardPro
      title={t('分类列表')}
      actionsArea={
        <Button theme='solid' icon={<Plus size={14} />} onClick={() => openEdit()}>
          {t('新增')}
        </Button>
      }
      paginationArea={
        <div className='flex justify-end'>
          <span className='text-sm text-gray-500'>
            {t('共 {{total}} 条', { total })}
          </span>
        </div>
      }
    >
      <CardTable
        loading={loading}
        columns={columns}
        dataSource={categories}
        pagination={{
          currentPage: activePage,
          pageSize,
          total,
          onPageChange: (page) => {
            setActivePage(page);
            loadCategories(page, pageSize);
          },
          onPageSizeChange: (size) => {
            setPageSize(size);
            setActivePage(1);
            loadCategories(1, size);
          },
        }}
      />

      <SideSheet
        title={editTitle}
        visible={showEdit}
        onCancel={() => setShowEdit(false)}
        width={450}
      >
        <Form
          initValues={initValues}
          getFormApi={setFormApi}
        >
          <Form.Input
            field='name'
            label={t('名称')}
            rules={[{ required: true, message: t('名称不能为空') }]}
          />
          <Form.TextArea field='description' label={t('描述')} rows={2} />
          <Form.Input field='icon' label={t('图标 (Lucide icon 名称)')} placeholder='BookOpen' />
          <Form.InputNumber field='sort_order' label={t('排序权重')} />
          <Form.Switch field='status' label={t('启用')} />
          <div className='flex justify-end gap-2 mt-4'>
            <Button theme='light' onClick={() => setShowEdit(false)}>
              {t('取消')}
            </Button>
            <Button theme='solid' onClick={handleSave}>
              {t('保存')}
            </Button>
          </div>
        </Form>
      </SideSheet>
    </CardPro>
  );
}
