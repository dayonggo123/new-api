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

const { Text } = Typography;
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsEchotik(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    'echotik_setting.echotik_enabled': false,
    'echotik_setting.echotik_base_url': 'https://open.echotik.live',
    'echotik_setting.echotik_username': '',
    'echotik_setting.echotik_password': '',
  });
  const [passwordSet, setPasswordSet] = useState(false);
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((inputs) => ({ ...inputs, [fieldName]: value }));
    };
  }

  function onSubmit() {
    let updateArray = compareObjects(inputs, inputsRow);
    // 如果密码为空字符串且之前已设置，不提交（避免覆盖已保存的密钥）
    updateArray = updateArray.filter((item) => {
      if (item.key === 'echotik_setting.echotik_password' && inputs[item.key] === '') {
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
      'echotik_setting.echotik_enabled': false,
      'echotik_setting.echotik_base_url': 'https://open.echotik.live',
      'echotik_setting.echotik_username': '',
      'echotik_setting.echotik_password': '',
    };
    const currentInputs = {};
    let pwdSet = false;
    for (const key in defaults) {
      let value = props.options[key];
      if (key === 'echotik_setting.echotik_enabled') {
        value = value === 'true' || value === true;
      } else if (key === 'echotik_setting.echotik_password') {
        if (value === '******') {
          pwdSet = true;
          value = '';
        } else {
          value = value !== undefined && value !== null ? String(value) : defaults[key];
        }
      } else {
        value = value !== undefined && value !== null ? String(value) : defaults[key];
      }
      currentInputs[key] = value;
    }
    setPasswordSet(pwdSet);
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
          <Form.Section text={t('EchoTik 设置')}>
            <div style={{ marginBottom: 16, padding: 12, background: '#f0f9ff', borderRadius: 8, borderLeft: '4px solid #0ea5e9' }}>
              <Text type='secondary' style={{ fontSize: 13, lineHeight: 1.6 }}>
                <strong>功能介绍：</strong>配置 EchoTik API 认证信息后，下游应用可通过 <code>/api/public/echotik/video/ranklist</code> 调用视频榜单接口，参数原样透传给 EchoTik。<br/>
                <strong>下游鉴权：</strong>调用新接口时需在 Header 传入 new-api 的 API Key（Authorization: Bearer sk-xxx）。
              </Text>
            </div>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'echotik_setting.echotik_enabled'}
                  label={t('启用 EchoTik 接口代理')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('echotik_setting.echotik_enabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'echotik_setting.echotik_base_url'}
                  label={t('EchoTik API 基础地址')}
                  initValue={'https://open.echotik.live'}
                  placeholder={t('例如 https://open.echotik.live')}
                  onChange={handleFieldChange('echotik_setting.echotik_base_url')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'echotik_setting.echotik_username'}
                  label={t('EchoTik 用户名')}
                  initValue={''}
                  placeholder={t('从 EchoTik 平台获取的 username')}
                  onChange={handleFieldChange('echotik_setting.echotik_username')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'echotik_setting.echotik_password'}
                  label={
                    <span>
                      {t('EchoTik 密码')}
                      {passwordSet && (
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
                    passwordSet
                      ? t('输入新值以覆盖已保存的密码')
                      : t('从 EchoTik 平台获取的 password')
                  }
                  onChange={handleFieldChange('echotik_setting.echotik_password')}
                  type='password'
                  showClear
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存 EchoTik 设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
