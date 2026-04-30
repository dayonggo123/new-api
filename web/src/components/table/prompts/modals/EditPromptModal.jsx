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

import React, { useEffect, useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  verifyJSONPromise,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Form,
  Avatar,
  Row,
  Col,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconClose,
  IconBook,
} from '@douyinfe/semi-icons';

const { Text, Title } = Typography;

const EditPromptModal = (props) => {
  const { t } = useTranslation();
  const isEdit = props.editingPrompt.id !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);

  const getInitValues = () => ({
    title: '',
    content: '',
    description: '',
    category_id: '',
    variables: '',
    tags: '',
    sort_order: 0,
    status: true,
  });

  const handleCancel = () => {
    props.handleClose();
  };

  const loadPrompt = async () => {
    setLoading(true);
    try {
      let res = await API.get(`/api/prompt/${props.editingPrompt.id}`);
      const { success, message, data } = res.data;
      if (success) {
        const values = {
          ...data,
          status: data.status === 1,
          variables: data.variables ? JSON.stringify(data.variables) : '',
          tags: data.tags ? JSON.stringify(data.tags) : '',
        };
        formApiRef.current?.setValues({ ...getInitValues(), ...values });
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        loadPrompt();
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [props.editingPrompt.id]);

  const submit = async (values) => {
    setLoading(true);
    let localInputs = { ...values };

    // Convert status boolean to number
    localInputs.status = localInputs.status ? 1 : 2;

    // Parse JSON fields
    if (localInputs.variables && localInputs.variables.trim() !== '') {
      try {
        localInputs.variables = JSON.parse(localInputs.variables);
      } catch (e) {
        showError(t('变量格式不正确，请输入合法的JSON'));
        setLoading(false);
        return;
      }
    } else {
      localInputs.variables = [];
    }

    if (localInputs.tags && localInputs.tags.trim() !== '') {
      try {
        localInputs.tags = JSON.parse(localInputs.tags);
      } catch (e) {
        showError(t('标签格式不正确，请输入合法的JSON'));
        setLoading(false);
        return;
      }
    } else {
      localInputs.tags = [];
    }

    localInputs.sort_order = parseInt(localInputs.sort_order) || 0;

    let res;
    try {
      if (isEdit) {
        res = await API.put(`/api/prompt/`, {
          ...localInputs,
          id: parseInt(props.editingPrompt.id),
        });
      } else {
        res = await API.post(`/api/prompt/`, {
          ...localInputs,
        });
      }
      const { success, message } = res.data;
      if (success) {
        if (isEdit) {
          showSuccess(t('提示词更新成功！'));
        } else {
          showSuccess(t('提示词创建成功！'));
        }
        props.refresh();
        props.handleClose();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  return (
    <>
      <SideSheet
        placement={isEdit ? 'right' : 'left'}
        title={
          <Space>
            {isEdit ? (
              <Tag color='blue' shape='circle'>
                {t('更新')}
              </Tag>
            ) : (
              <Tag color='green' shape='circle'>
                {t('新建')}
              </Tag>
            )}
            <Title heading={4} className='m-0'>
              {isEdit ? t('更新提示词信息') : t('创建新的提示词')}
            </Title>
          </Space>
        }
        bodyStyle={{ padding: '0' }}
        visible={props.visiable}
        width={isMobile ? '100%' : 600}
        footer={
          <div className='flex justify-end bg-white'>
            <Space>
              <Button
                theme='solid'
                onClick={() => formApiRef.current?.submitForm()}
                icon={<IconSave />}
                loading={loading}
              >
                {t('提交')}
              </Button>
              <Button
                theme='light'
                type='primary'
                onClick={handleCancel}
                icon={<IconClose />}
              >
                {t('取消')}
              </Button>
            </Space>
          </div>
        }
        closeIcon={null}
        onCancel={() => handleCancel()}
      >
        <Spin spinning={loading}>
          <Form
            initValues={getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
          >
            {() => (
              <div className='p-2'>
                <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
                  {/* Header: Basic Info */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='blue'
                      className='mr-2 shadow-md'
                    >
                      <IconBook size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('基本信息')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置提示词的基本信息')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input
                        field='title'
                        label={t('标题')}
                        placeholder={t('请输入标题')}
                        style={{ width: '100%' }}
                        rules={[
                          { required: true, message: t('请输入标题') },
                        ]}
                        showClear
                      />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea
                        field='content'
                        label={t('内容')}
                        placeholder={t('请输入提示词内容')}
                        rows={5}
                        style={{ width: '100%' }}
                        rules={[
                          { required: true, message: t('请输入内容') },
                        ]}
                      />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea
                        field='description'
                        label={t('描述')}
                        placeholder={t('请输入描述')}
                        rows={2}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col span={24}>
                      <Form.Select
                        field='category_id'
                        label={t('分类')}
                        placeholder={t('请选择分类')}
                        style={{ width: '100%' }}
                        rules={[
                          { required: true, message: t('请选择分类') },
                        ]}
                      >
                        {props.categories?.map((cat) => (
                          <Form.Select.Option key={cat.id} value={cat.id}>
                            {cat.name}
                          </Form.Select.Option>
                        ))}
                      </Form.Select>
                    </Col>
                  </Row>
                </Card>

                <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
                  {/* Header: Advanced Settings */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='green'
                      className='mr-2 shadow-md'
                    >
                      <IconBook size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('高级设置')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置提示词的高级属性')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.TextArea
                        field='variables'
                        label={t('变量')}
                        placeholder={t('JSON格式 [{"name":"subject","label":"主题"}]')}
                        rows={2}
                        style={{ width: '100%' }}
                        rules={[
                          {
                            validator: (rule, value) => {
                              if (!value || value.trim() === '') {
                                return Promise.resolve();
                              }
                              return verifyJSONPromise(value);
                            },
                            message: t('请输入合法的JSON格式'),
                          },
                        ]}
                      />
                    </Col>
                    <Col span={24}>
                      <Form.TextArea
                        field='tags'
                        label={t('标签')}
                        placeholder={t('JSON格式 ["科幻", "风景"]')}
                        rows={2}
                        style={{ width: '100%' }}
                        rules={[
                          {
                            validator: (rule, value) => {
                              if (!value || value.trim() === '') {
                                return Promise.resolve();
                              }
                              return verifyJSONPromise(value);
                            },
                            message: t('请输入合法的JSON格式'),
                          },
                        ]}
                      />
                    </Col>
                    <Col span={12}>
                      <Form.InputNumber
                        field='sort_order'
                        label={t('排序')}
                        placeholder={t('请输入排序值')}
                        min={0}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col span={12}>
                      <div className='flex items-center h-full pt-6'>
                        <Form.Switch
                          field='status'
                          label={t('状态')}
                          checkedText={t('启用')}
                          uncheckedText={t('禁用')}
                        />
                      </div>
                    </Col>
                  </Row>
                </Card>
              </div>
            )}
          </Form>
        </Spin>
      </SideSheet>
    </>
  );
};

export default EditPromptModal;
