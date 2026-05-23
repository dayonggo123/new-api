/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect, useRef } from 'react';
import {
  Tabs,
  Button,
  Card,
  Form,
  Input,
  Row,
  Col,
  Tag,
  Space,
  Spin,
  Typography,
  SideSheet,
  Table,
  Popconfirm,
  Switch,
  Avatar,
  Pagination,
  TextArea,
  Select,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconClose,
  IconPlus,
  IconEdit,
  IconRefresh,
  IconImage,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

const { Text, Title } = Typography;

// ==================== Utility ====================

const parseJsonLines = (value) => {
  if (!value) return [];
  if (Array.isArray(value)) return value;
  try {
    const parsed = JSON.parse(value);
    if (Array.isArray(parsed)) return parsed;
    return [];
  } catch (e) {
    return value.split('\n').filter((line) => line.trim() !== '');
  }
};

const formatJsonLines = (value) => {
  if (!value) return '';
  if (Array.isArray(value)) return value.join('\n');
  try {
    const parsed = JSON.parse(value);
    if (Array.isArray(parsed)) return parsed.join('\n');
    return String(value);
  } catch (e) {
    return String(value);
  }
};

// ==================== ModelPose Edit SideSheet ====================

const ModelPoseEditSheet = ({ visible, onCancel, record, refresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isEdit = record?.id !== undefined;

  const getInitValues = () => ({
    pose_id: '',
    label: '',
    description: '',
    cover_image_url: '',
    sort_order: 0,
    status: true,
  });

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        formApiRef.current.setValues({
          ...record,
          status: record.status === 1,
        });
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [record?.id, visible]);

  const submit = async (values) => {
    setLoading(true);
    const payload = {
      ...values,
      status: values.status ? 1 : 2,
      sort_order: parseInt(values.sort_order) || 0,
    };
    try {
      let res;
      if (isEdit) {
        res = await API.put(`/api/admin/ecommerce/model-poses/${record.id}`, payload);
      } else {
        res = await API.post('/api/admin/ecommerce/model-poses', payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(isEdit ? t('更新成功') : t('创建成功'));
        await refresh();
        onCancel();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  return (
    <SideSheet
      title={
        <Space>
          {isEdit ? (
            <Tag color='blue' shape='circle'>{t('更新')}</Tag>
          ) : (
            <Tag color='green' shape='circle'>{t('新建')}</Tag>
          )}
          <Title heading={4} className='m-0'>
            {isEdit ? t('编辑模特呈现方式') : t('新建模特呈现方式')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={560}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button theme='solid' onClick={() => formApiRef.current?.submitForm()} icon={<IconSave />} loading={loading}>
              {t('提交')}
            </Button>
            <Button theme='light' type='primary' onClick={onCancel} icon={<IconClose />}>
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={onCancel}
    >
      <Spin spinning={loading}>
        <Form initValues={getInitValues()} getFormApi={(api) => (formApiRef.current = api)} onSubmit={submit}>
          {() => (
            <div className='p-4'>
              <Card className='!rounded-2xl shadow-sm border-0'>
                <Row gutter={12}>
                  <Col span={12}>
                    <Form.Input field='pose_id' label={t('标识')} placeholder={t('如 no_model, hold_product')} rules={[{ required: true, message: t('请输入标识') }]} showClear />
                  </Col>
                  <Col span={12}>
                    <Form.Input field='label' label={t('显示名称')} placeholder={t('请输入显示名称')} rules={[{ required: true, message: t('请输入显示名称') }]} showClear />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea field='description' label={t('描述')} placeholder={t('请输入描述')} rows={2} />
                  </Col>
                  <Col span={24}>
                    <Form.Input field='cover_image_url' label={t('封面图 URL')} placeholder={t('请输入封面图地址')} showClear />
                  </Col>
                  <Col span={12}>
                    <Form.InputNumber field='sort_order' label={t('排序')} placeholder={t('请输入排序值')} min={0} />
                  </Col>
                  <Col span={12}>
                    <div className='flex items-center h-full pt-6'>
                      <Form.Switch field='status' label={t('状态')} checkedText={t('启用')} uncheckedText={t('禁用')} />
                    </div>
                  </Col>
                </Row>
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

// ==================== CaseCategory Edit SideSheet ====================

const CaseCategoryEditSheet = ({ visible, onCancel, record, refresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isEdit = record?.id !== undefined;

  const getInitValues = () => ({
    category_id: '',
    category_name: '',
    cover_image_url: '',
    requires_model: false,
    sort_order: 0,
    status: true,
  });

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        formApiRef.current.setValues({
          ...record,
          status: record.status === 1,
          requires_model: record.requires_model === true || record.requires_model === 1,
        });
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [record?.id, visible]);

  const submit = async (values) => {
    setLoading(true);
    const payload = {
      ...values,
      status: values.status ? 1 : 2,
      requires_model: values.requires_model ? true : false,
      sort_order: parseInt(values.sort_order) || 0,
    };
    try {
      let res;
      if (isEdit) {
        res = await API.put(`/api/admin/ecommerce/case-categories/${record.id}`, payload);
      } else {
        res = await API.post('/api/admin/ecommerce/case-categories', payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(isEdit ? t('更新成功') : t('创建成功'));
        await refresh();
        onCancel();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  return (
    <SideSheet
      title={
        <Space>
          {isEdit ? (
            <Tag color='blue' shape='circle'>{t('更新')}</Tag>
          ) : (
            <Tag color='green' shape='circle'>{t('新建')}</Tag>
          )}
          <Title heading={4} className='m-0'>
            {isEdit ? t('编辑案例库品类') : t('新建案例库品类')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={560}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button theme='solid' onClick={() => formApiRef.current?.submitForm()} icon={<IconSave />} loading={loading}>
              {t('提交')}
            </Button>
            <Button theme='light' type='primary' onClick={onCancel} icon={<IconClose />}>
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={onCancel}
    >
      <Spin spinning={loading}>
        <Form initValues={getInitValues()} getFormApi={(api) => (formApiRef.current = api)} onSubmit={submit}>
          {() => (
            <div className='p-4'>
              <Card className='!rounded-2xl shadow-sm border-0'>
                <Row gutter={12}>
                  <Col span={12}>
                    <Form.Input field='category_id' label={t('品类标识')} placeholder={t('如 clothing, electronics')} rules={[{ required: true, message: t('请输入品类标识') }]} showClear />
                  </Col>
                  <Col span={12}>
                    <Form.Input field='category_name' label={t('品类名称')} placeholder={t('请输入品类名称')} rules={[{ required: true, message: t('请输入品类名称') }]} showClear />
                  </Col>
                  <Col span={24}>
                    <Form.Input field='cover_image_url' label={t('封面图 URL')} placeholder={t('请输入封面图地址')} showClear />
                  </Col>
                  <Col span={12}>
                    <Form.InputNumber field='sort_order' label={t('排序')} placeholder={t('请输入排序值')} min={0} />
                  </Col>
                  <Col span={12}>
                    <div className='flex items-center h-full pt-6'>
                      <Form.Switch field='requires_model' label={t('需要模特')} checkedText={t('是')} uncheckedText={t('否')} />
                    </div>
                  </Col>
                  <Col span={24}>
                    <div className='flex items-center h-full pt-2'>
                      <Form.Switch field='status' label={t('状态')} checkedText={t('启用')} uncheckedText={t('禁用')} />
                    </div>
                  </Col>
                </Row>
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

// ==================== CaseDetail Edit SideSheet ====================

const CaseDetailEditSheet = ({ visible, onCancel, record, refresh, categoryOptions }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isEdit = record?.id !== undefined;

  const getInitValues = () => ({
    category_id: '',
    platform_id: '',
    platform_name: '',
    visual_features: '',
    composition: '',
    lighting: '',
    background_style: '',
    case_reference: '',
  });

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        formApiRef.current.setValues({
          ...record,
          visual_features: formatJsonLines(record.visual_features),
          composition: formatJsonLines(record.composition),
        });
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [record?.id, visible]);

  const submit = async (values) => {
    setLoading(true);
    const payload = {
      ...values,
      visual_features: JSON.stringify(
        values.visual_features
          ? values.visual_features.split('\n').filter((line) => line.trim() !== '')
          : []
      ),
      composition: JSON.stringify(
        values.composition
          ? values.composition.split('\n').filter((line) => line.trim() !== '')
          : []
      ),
    };
    try {
      let res;
      if (isEdit) {
        res = await API.put(`/api/admin/ecommerce/case-details/${record.id}`, payload);
      } else {
        res = await API.post('/api/admin/ecommerce/case-details', payload);
      }
      const { success, message } = res.data;
      if (success) {
        showSuccess(isEdit ? t('更新成功') : t('创建成功'));
        await refresh();
        onCancel();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  return (
    <SideSheet
      title={
        <Space>
          {isEdit ? (
            <Tag color='blue' shape='circle'>{t('更新')}</Tag>
          ) : (
            <Tag color='green' shape='circle'>{t('新建')}</Tag>
          )}
          <Title heading={4} className='m-0'>
            {isEdit ? t('编辑案例库详情') : t('新建案例库详情')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={visible}
      width={600}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button theme='solid' onClick={() => formApiRef.current?.submitForm()} icon={<IconSave />} loading={loading}>
              {t('提交')}
            </Button>
            <Button theme='light' type='primary' onClick={onCancel} icon={<IconClose />}>
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={onCancel}
    >
      <Spin spinning={loading}>
        <Form initValues={getInitValues()} getFormApi={(api) => (formApiRef.current = api)} onSubmit={submit}>
          {() => (
            <div className='p-4'>
              <Card className='!rounded-2xl shadow-sm border-0'>
                <Row gutter={12}>
                  <Col span={12}>
                    <Form.Select
                      field='category_id'
                      label={t('品类')}
                      placeholder={t('请选择品类')}
                      rules={[{ required: true, message: t('请选择品类') }]}
                      optionList={categoryOptions}
                      style={{ width: '100%' }}
                      filter
                    />
                  </Col>
                  <Col span={12}>
                    <Form.Input field='platform_id' label={t('平台标识')} placeholder={t('如 taobao, jd')} rules={[{ required: true, message: t('请输入平台标识') }]} showClear />
                  </Col>
                  <Col span={24}>
                    <Form.Input field='platform_name' label={t('平台名称')} placeholder={t('如淘宝、京东')} showClear />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea
                      field='visual_features'
                      label={t('视觉特征')}
                      placeholder={t('每行一条，保存为 JSON 数组')}
                      rows={3}
                    />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea
                      field='composition'
                      label={t('构图方式')}
                      placeholder={t('每行一条，保存为 JSON 数组')}
                      rows={3}
                    />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea field='lighting' label={t('光线风格')} placeholder={t('请输入光线风格描述')} rows={2} />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea field='background_style' label={t('背景风格')} placeholder={t('请输入背景风格描述')} rows={2} />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea field='case_reference' label={t('案例参考')} placeholder={t('请输入案例参考信息')} rows={3} />
                  </Col>
                </Row>
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

// ==================== Main Page ====================

const EcommerceWizardManagement = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('model_poses');

  // Model Poses state
  const [modelPoses, setModelPoses] = useState([]);
  const [modelPoseTotal, setModelPoseTotal] = useState(0);
  const [modelPosePage, setModelPosePage] = useState(1);
  const [modelPosePageSize, setModelPosePageSize] = useState(ITEMS_PER_PAGE);
  const [modelPoseLoading, setModelPoseLoading] = useState(false);
  const [editingModelPose, setEditingModelPose] = useState(null);
  const [showModelPoseEdit, setShowModelPoseEdit] = useState(false);

  // Case Categories state
  const [caseCategories, setCaseCategories] = useState([]);
  const [caseCategoryTotal, setCaseCategoryTotal] = useState(0);
  const [caseCategoryPage, setCaseCategoryPage] = useState(1);
  const [caseCategoryPageSize, setCaseCategoryPageSize] = useState(ITEMS_PER_PAGE);
  const [caseCategoryLoading, setCaseCategoryLoading] = useState(false);
  const [editingCaseCategory, setEditingCaseCategory] = useState(null);
  const [showCaseCategoryEdit, setShowCaseCategoryEdit] = useState(false);

  // Case Details state
  const [caseDetails, setCaseDetails] = useState([]);
  const [caseDetailTotal, setCaseDetailTotal] = useState(0);
  const [caseDetailPage, setCaseDetailPage] = useState(1);
  const [caseDetailPageSize, setCaseDetailPageSize] = useState(ITEMS_PER_PAGE);
  const [caseDetailLoading, setCaseDetailLoading] = useState(false);
  const [editingCaseDetail, setEditingCaseDetail] = useState(null);
  const [showCaseDetailEdit, setShowCaseDetailEdit] = useState(false);

  const categoryOptions = caseCategories.map((c) => ({
    label: c.category_name || c.category_id,
    value: c.category_id,
  }));

  const loadModelPoses = async (page = modelPosePage, pageSize = modelPosePageSize) => {
    setModelPoseLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('p', page);
      params.append('page_size', pageSize);
      const res = await API.get(`/api/admin/ecommerce/model-poses?${params.toString()}`);
      const { success, message, data } = res.data;
      if (success && data) {
        setModelPoses(data.items || data || []);
        setModelPoseTotal(data.total || (data.items ? data.items.length : data.length) || 0);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setModelPoseLoading(false);
  };

  const loadCaseCategories = async (page = caseCategoryPage, pageSize = caseCategoryPageSize) => {
    setCaseCategoryLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('p', page);
      params.append('page_size', pageSize);
      const res = await API.get(`/api/admin/ecommerce/case-categories?${params.toString()}`);
      const { success, message, data } = res.data;
      if (success && data) {
        setCaseCategories(data.items || data || []);
        setCaseCategoryTotal(data.total || (data.items ? data.items.length : data.length) || 0);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setCaseCategoryLoading(false);
  };

  const loadCaseDetails = async (page = caseDetailPage, pageSize = caseDetailPageSize) => {
    setCaseDetailLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('p', page);
      params.append('page_size', pageSize);
      const res = await API.get(`/api/admin/ecommerce/case-details?${params.toString()}`);
      const { success, message, data } = res.data;
      if (success && data) {
        setCaseDetails(data.items || data || []);
        setCaseDetailTotal(data.total || (data.items ? data.items.length : data.length) || 0);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setCaseDetailLoading(false);
  };

  useEffect(() => {
    loadModelPoses();
  }, []);

  useEffect(() => {
    loadCaseCategories();
  }, []);

  useEffect(() => {
    loadCaseDetails();
  }, []);

  const handleDeleteModelPose = async (id) => {
    try {
      const res = await API.delete(`/api/admin/ecommerce/model-poses/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('删除成功'));
        loadModelPoses();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleDeleteCaseCategory = async (id) => {
    try {
      const res = await API.delete(`/api/admin/ecommerce/case-categories/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('删除成功'));
        loadCaseCategories();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleDeleteCaseDetail = async (id) => {
    try {
      const res = await API.delete(`/api/admin/ecommerce/case-details/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('删除成功'));
        loadCaseDetails();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const getCategoryName = (categoryId) => {
    const cat = caseCategories.find((c) => c.category_id === categoryId);
    return cat ? cat.category_name : categoryId || '-';
  };

  const modelPoseColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('标识'),
      dataIndex: 'pose_id',
      width: 140,
    },
    {
      title: t('显示名称'),
      dataIndex: 'label',
      render: (text, record) => (
        <div className='flex items-center gap-2'>
          {record.cover_image_url && (
            <img src={record.cover_image_url} alt='' className='w-8 h-8 rounded object-cover' />
          )}
          <Text strong>{text}</Text>
        </div>
      ),
    },
    {
      title: t('描述'),
      dataIndex: 'description',
      render: (text) => text || '-',
    },
    {
      title: t('排序'),
      dataIndex: 'sort_order',
      width: 80,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'red'} shape='circle'>
          {status === 1 ? t('启用') : t('禁用')}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      fixed: 'right',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button type='tertiary' size='small' icon={<IconEdit />} onClick={() => {
            setEditingModelPose(record);
            setShowModelPoseEdit(true);
          }}>
            {t('编辑')}
          </Button>
          <Popconfirm title={t('确定删除此呈现方式吗？')} content={t('此操作不可撤销')} onConfirm={() => handleDeleteModelPose(record.id)}>
            <Button type='danger' theme='light' size='small'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const caseCategoryColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('品类标识'),
      dataIndex: 'category_id',
      width: 140,
    },
    {
      title: t('品类名称'),
      dataIndex: 'category_name',
      render: (text, record) => (
        <div className='flex items-center gap-2'>
          {record.cover_image_url && (
            <img src={record.cover_image_url} alt='' className='w-8 h-8 rounded object-cover' />
          )}
          <Text strong>{text}</Text>
        </div>
      ),
    },
    {
      title: t('需要模特'),
      dataIndex: 'requires_model',
      width: 100,
      render: (v) => (
        <Tag color={v ? 'green' : 'grey'} shape='circle'>
          {v ? t('是') : t('否')}
        </Tag>
      ),
    },
    {
      title: t('排序'),
      dataIndex: 'sort_order',
      width: 80,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'red'} shape='circle'>
          {status === 1 ? t('启用') : t('禁用')}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      fixed: 'right',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button type='tertiary' size='small' icon={<IconEdit />} onClick={() => {
            setEditingCaseCategory(record);
            setShowCaseCategoryEdit(true);
          }}>
            {t('编辑')}
          </Button>
          <Popconfirm title={t('确定删除此品类吗？')} content={t('此操作不可撤销')} onConfirm={() => handleDeleteCaseCategory(record.id)}>
            <Button type='danger' theme='light' size='small'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const caseDetailColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('品类'),
      dataIndex: 'category_id',
      width: 120,
      render: (text) => getCategoryName(text),
    },
    {
      title: t('平台'),
      dataIndex: 'platform_name',
      width: 120,
      render: (text, record) => (
        <div>
          <Text strong>{text || record.platform_id}</Text>
          {text && text !== record.platform_id && (
            <Text type='tertiary' size='small' className='block'>{record.platform_id}</Text>
          )}
        </div>
      ),
    },
    {
      title: t('视觉特征'),
      dataIndex: 'visual_features',
      render: (text) => {
        const items = parseJsonLines(text);
        if (items.length === 0) return '-';
        return (
          <div className='flex flex-wrap gap-1'>
            {items.slice(0, 3).map((item, idx) => (
              <Tag key={idx} size='small' color='light-blue'>{item}</Tag>
            ))}
            {items.length > 3 && <Tag size='small' color='grey'>+{items.length - 3}</Tag>}
          </div>
        );
      },
    },
    {
      title: t('光线风格'),
      dataIndex: 'lighting',
      width: 140,
      render: (text) => text || '-',
    },
    {
      title: t('背景风格'),
      dataIndex: 'background_style',
      width: 140,
      render: (text) => text || '-',
    },
    {
      title: t('操作'),
      fixed: 'right',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button type='tertiary' size='small' icon={<IconEdit />} onClick={() => {
            setEditingCaseDetail(record);
            setShowCaseDetailEdit(true);
          }}>
            {t('编辑')}
          </Button>
          <Popconfirm title={t('确定删除此案例详情吗？')} content={t('此操作不可撤销')} onConfirm={() => handleDeleteCaseDetail(record.id)}>
            <Button type='danger' theme='light' size='small'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <div style={{ marginBottom: 16, padding: 12, background: '#f0f9ff', borderRadius: 8, borderLeft: '4px solid #0ea5e9' }}>
        <Text type='secondary' style={{ fontSize: 13, lineHeight: 1.6 }}>
          <strong>{t('功能介绍：')}</strong>{t('管理电商图向导的模特呈现方式和案例库配置')}
        </Text>
      </div>

      <ModelPoseEditSheet
        visible={showModelPoseEdit}
        onCancel={() => {
          setShowModelPoseEdit(false);
          setEditingModelPose(null);
        }}
        record={editingModelPose}
        refresh={loadModelPoses}
      />

      <CaseCategoryEditSheet
        visible={showCaseCategoryEdit}
        onCancel={() => {
          setShowCaseCategoryEdit(false);
          setEditingCaseCategory(null);
        }}
        record={editingCaseCategory}
        refresh={loadCaseCategories}
      />

      <CaseDetailEditSheet
        visible={showCaseDetailEdit}
        onCancel={() => {
          setShowCaseDetailEdit(false);
          setEditingCaseDetail(null);
        }}
        record={editingCaseDetail}
        refresh={loadCaseDetails}
        categoryOptions={categoryOptions}
      />

      <Tabs type='line' activeKey={activeTab} onChange={(key) => setActiveTab(key)}>
        <Tabs.TabPane tab={t('模特呈现方式')} itemKey='model_poses'>
          <Card className='!rounded-2xl shadow-sm border-0'>
            <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-3 mb-4'>
              <div className='flex items-center gap-2'>
                <Avatar size='small' color='blue'>
                  <IconImage size={16} />
                </Avatar>
                <Text className='text-lg font-medium'>{t('模特呈现方式列表')}</Text>
              </div>
              <Button type='primary' size='small' icon={<IconPlus />} onClick={() => {
                setEditingModelPose(null);
                setShowModelPoseEdit(true);
              }}>
                {t('新增呈现方式')}
              </Button>
            </div>

            <Spin spinning={modelPoseLoading}>
              <Table
                columns={modelPoseColumns}
                dataSource={modelPoses}
                pagination={false}
                emptyText={t('暂无数据')}
                size='small'
              />
              <div className='flex justify-end mt-4'>
                <Pagination
                  total={modelPoseTotal}
                  pageSize={modelPosePageSize}
                  currentPage={modelPosePage}
                  onPageChange={(page) => {
                    setModelPosePage(page);
                    loadModelPoses(page, modelPosePageSize);
                  }}
                  showSizeChanger
                  pageSizeOpts={[10, 20, 50, 100]}
                  onShowSizeChange={(current, size) => {
                    setModelPosePageSize(size);
                    setModelPosePage(1);
                    loadModelPoses(1, size);
                  }}
                />
              </div>
            </Spin>
          </Card>
        </Tabs.TabPane>

        <Tabs.TabPane tab={t('案例库品类')} itemKey='case_categories'>
          <Card className='!rounded-2xl shadow-sm border-0'>
            <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-3 mb-4'>
              <div className='flex items-center gap-2'>
                <Avatar size='small' color='blue'>
                  <IconImage size={16} />
                </Avatar>
                <Text className='text-lg font-medium'>{t('案例库品类列表')}</Text>
              </div>
              <Button type='primary' size='small' icon={<IconPlus />} onClick={() => {
                setEditingCaseCategory(null);
                setShowCaseCategoryEdit(true);
              }}>
                {t('新增品类')}
              </Button>
            </div>

            <Spin spinning={caseCategoryLoading}>
              <Table
                columns={caseCategoryColumns}
                dataSource={caseCategories}
                pagination={false}
                emptyText={t('暂无数据')}
                size='small'
              />
              <div className='flex justify-end mt-4'>
                <Pagination
                  total={caseCategoryTotal}
                  pageSize={caseCategoryPageSize}
                  currentPage={caseCategoryPage}
                  onPageChange={(page) => {
                    setCaseCategoryPage(page);
                    loadCaseCategories(page, caseCategoryPageSize);
                  }}
                  showSizeChanger
                  pageSizeOpts={[10, 20, 50, 100]}
                  onShowSizeChange={(current, size) => {
                    setCaseCategoryPageSize(size);
                    setCaseCategoryPage(1);
                    loadCaseCategories(1, size);
                  }}
                />
              </div>
            </Spin>
          </Card>
        </Tabs.TabPane>

        <Tabs.TabPane tab={t('案例库详情')} itemKey='case_details'>
          <Card className='!rounded-2xl shadow-sm border-0'>
            <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-3 mb-4'>
              <div className='flex items-center gap-2'>
                <Avatar size='small' color='blue'>
                  <IconImage size={16} />
                </Avatar>
                <Text className='text-lg font-medium'>{t('案例库详情列表')}</Text>
              </div>
              <Button type='primary' size='small' icon={<IconPlus />} onClick={() => {
                setEditingCaseDetail(null);
                setShowCaseDetailEdit(true);
              }}>
                {t('新增案例详情')}
              </Button>
            </div>

            <Spin spinning={caseDetailLoading}>
              <Table
                columns={caseDetailColumns}
                dataSource={caseDetails}
                pagination={false}
                emptyText={t('暂无数据')}
                size='small'
              />
              <div className='flex justify-end mt-4'>
                <Pagination
                  total={caseDetailTotal}
                  pageSize={caseDetailPageSize}
                  currentPage={caseDetailPage}
                  onPageChange={(page) => {
                    setCaseDetailPage(page);
                    loadCaseDetails(page, caseDetailPageSize);
                  }}
                  showSizeChanger
                  pageSizeOpts={[10, 20, 50, 100]}
                  onShowSizeChange={(current, size) => {
                    setCaseDetailPageSize(size);
                    setCaseDetailPage(1);
                    loadCaseDetails(1, size);
                  }}
                />
              </div>
            </Spin>
          </Card>
        </Tabs.TabPane>
      </Tabs>
    </div>
  );
};

export default EcommerceWizardManagement;
