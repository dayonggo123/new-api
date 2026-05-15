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
import { Button, Col, Form, Row, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsImageAI(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    'image_ai_setting.image_ai_enabled': false,
    'image_ai_setting.image_ai_model': 'dall-e-3',
    'image_ai_setting.image_ai_base_url': '',
    'image_ai_setting.image_ai_api_key': '',
    'image_ai_setting.image_ai_size': '1024x1024',
    'image_ai_setting.image_ai_n': '1',
  });
  const [apiKeySet, setApiKeySet] = useState(false);
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((inputs) => ({ ...inputs, [fieldName]: value }));
    };
  }

  function onSubmit() {
    let updateArray = compareObjects(inputs, inputsRow);
    updateArray = updateArray.filter((item) => {
      if (item.key === 'image_ai_setting.image_ai_api_key' && inputs[item.key] === '') {
        return false;
      }
      return true;
    });
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = String(inputs[item.key]);
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const defaults = {
      'image_ai_setting.image_ai_enabled': false,
      'image_ai_setting.image_ai_model': 'dall-e-3',
      'image_ai_setting.image_ai_base_url': '',
      'image_ai_setting.image_ai_api_key': '',
      'image_ai_setting.image_ai_size': '1024x1024',
      'image_ai_setting.image_ai_n': '1',
    };
    const currentInputs = {};
    let keySet = false;
    for (const key in defaults) {
      let value = props.options[key];
      if (key === 'image_ai_setting.image_ai_enabled') {
        value = value === 'true' || value === true;
      } else if (key === 'image_ai_setting.image_ai_api_key') {
        if (value === '******') {
          keySet = true;
          value = '';
        } else {
          value = value !== undefined && value !== null ? String(value) : defaults[key];
        }
      } else if (key === 'image_ai_setting.image_ai_n') {
        value = value !== undefined && value !== null ? String(value) : defaults[key];
      } else {
        value = value !== undefined && value !== null ? String(value) : defaults[key];
      }
      currentInputs[key] = value;
    }
    setApiKeySet(keySet);
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('文章图片 AI 设置')}>
            <div style={{ marginBottom: 16, padding: 12, background: '#f0f9ff', borderRadius: 8, borderLeft: '4px solid #0ea5e9' }}>
              <Text type='secondary' style={{ fontSize: 13, lineHeight: 1.6 }}>
                <strong>功能介绍：</strong>配置 AI 图片生成引擎，用于在文章编辑中自动生成封面图或正文配图。支持 DALL-E 格式的图片生成 API。<br/>
                <strong>如何修改：</strong>填写支持 /v1/images/generations 接口的服务地址和 Key（如 OpenAI、Azure OpenAI 等）。模型用 dall-e-3 或 gpt-image-1。尺寸和数量是默认值，编辑文章时可以在弹窗中临时调整。图片生成后会返回 URL，点击缩略图即可选用。
              </Text>
            </div>
            <Typography.Text
              type='tertiary'
              style={{ marginBottom: 16, display: 'block' }}
            >
              {t('启用后，可在文章编辑中调用 AI 生成封面图或正文配图')}
            </Typography.Text>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'image_ai_setting.image_ai_enabled'}
                  label={t('启用 AI 自动生成图片')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('image_ai_setting.image_ai_enabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'image_ai_setting.image_ai_model'}
                  label={t('AI 模型')}
                  initValue={'dall-e-3'}
                  placeholder={t('例如 dall-e-3')}
                  onChange={handleFieldChange('image_ai_setting.image_ai_model')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'image_ai_setting.image_ai_base_url'}
                  label={t('API Base URL')}
                  initValue={''}
                  placeholder={t('例如 https://api.openai.com/v1')}
                  onChange={handleFieldChange('image_ai_setting.image_ai_base_url')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'image_ai_setting.image_ai_api_key'}
                  label={
                    <span>
                      {t('API Key')}
                      {apiKeySet && (
                        <Tag
                          color='green'
                          size='small'
                          style={{ marginLeft: 8 }}
                        >
                          {t('已设置')}
                        </Tag>
                      )}
                    </span>
                  }
                  initValue={''}
                  placeholder={
                    apiKeySet
                      ? t('输入新值以覆盖已保存的密钥')
                      : t('输入 API Key')
                  }
                  onChange={handleFieldChange('image_ai_setting.image_ai_api_key')}
                  type='password'
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Select
                  field={'image_ai_setting.image_ai_size'}
                  label={t('默认尺寸')}
                  initValue={'1024x1024'}
                  onChange={handleFieldChange('image_ai_setting.image_ai_size')}
                >
                  <Form.Select.Option value='1024x1024'>1024x1024</Form.Select.Option>
                  <Form.Select.Option value='1024x1792'>1024x1792</Form.Select.Option>
                  <Form.Select.Option value='1792x1024'>1792x1024</Form.Select.Option>
                  <Form.Select.Option value='512x512'>512x512</Form.Select.Option>
                  <Form.Select.Option value='256x256'>256x256</Form.Select.Option>
                </Form.Select>
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'image_ai_setting.image_ai_n'}
                  label={t('默认生成数量')}
                  initValue={'1'}
                  placeholder={t('1-4')}
                  onChange={handleFieldChange('image_ai_setting.image_ai_n')}
                  showClear
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存图片 AI 设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
