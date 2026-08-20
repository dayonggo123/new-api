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
import { Button, Form, Row, Col, Typography, Spin, Banner } from '@douyinfe/semi-ui';
const { Text } = Typography;
import { API, showError, showSuccess, toBoolean } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const SENSITIVE_PLACEHOLDER = '******';

const SETTING_KEYS = {
  ENABLED: 'alipay_setting.alipay_enabled',
  APP_ID: 'alipay_setting.alipay_app_id',
  APP_PRIVATE_KEY: 'alipay_setting.alipay_app_private_key',
  PUBLIC_KEY: 'alipay_setting.alipay_public_key',
  AES_KEY: 'alipay_setting.alipay_aes_key',
  IS_SANDBOX: 'alipay_setting.alipay_is_sandbox',
};

const OPTION_KEYS = {
  AlipayEnabled: SETTING_KEYS.ENABLED,
  AlipayAppID: SETTING_KEYS.APP_ID,
  AlipayAppPrivateKey: SETTING_KEYS.APP_PRIVATE_KEY,
  AlipayPublicKey: SETTING_KEYS.PUBLIC_KEY,
  AlipayAESKey: SETTING_KEYS.AES_KEY,
  AlipayIsSandbox: SETTING_KEYS.IS_SANDBOX,
};

export default function SettingsPaymentGatewayAlipay(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AlipayEnabled: false,
    AlipayAppID: '',
    AlipayAppPrivateKey: '',
    AlipayPublicKey: '',
    AlipayAESKey: '',
    AlipayIsSandbox: false,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AlipayEnabled: toBoolean(props.options[OPTION_KEYS.AlipayEnabled]),
        AlipayAppID: props.options[OPTION_KEYS.AlipayAppID] || '',
        AlipayAppPrivateKey: props.options[OPTION_KEYS.AlipayAppPrivateKey] || '',
        AlipayPublicKey: props.options[OPTION_KEYS.AlipayPublicKey] || '',
        AlipayAESKey: props.options[OPTION_KEYS.AlipayAESKey] || '',
        AlipayIsSandbox: toBoolean(props.options[OPTION_KEYS.AlipayIsSandbox]),
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const isSensitiveUnchanged = (value) => {
    return value === SENSITIVE_PLACEHOLDER || value === '';
  };

  const submitAlipaySetting = async () => {
    if (props.options.ServerAddress === '') {
      showError(t('请先填写服务器地址'));
      return;
    }

    if (inputs.AlipayEnabled) {
      if (inputs.AlipayAppID === '') {
        showError('启用支付宝支付时必须填写 AppID');
        return;
      }
      const hasPrivateKey =
        !isSensitiveUnchanged(inputs.AlipayAppPrivateKey) ||
        !isSensitiveUnchanged(originInputs.AlipayAppPrivateKey);
      const hasPublicKey =
        !isSensitiveUnchanged(inputs.AlipayPublicKey) ||
        !isSensitiveUnchanged(originInputs.AlipayPublicKey);
      const hasAESKey =
        !isSensitiveUnchanged(inputs.AlipayAESKey) ||
        !isSensitiveUnchanged(originInputs.AlipayAESKey);
      if (!hasPrivateKey || !hasPublicKey || !hasAESKey) {
        showError('启用支付宝支付时必须填写应用私钥、支付宝公钥和 AES 密钥');
        return;
      }
    }

    setLoading(true);
    try {
      const options = [];

      options.push({
        key: SETTING_KEYS.ENABLED,
        value: inputs.AlipayEnabled ? 'true' : 'false',
      });
      options.push({
        key: SETTING_KEYS.IS_SANDBOX,
        value: inputs.AlipayIsSandbox ? 'true' : 'false',
      });

      if (inputs.AlipayAppID !== '') {
        options.push({ key: SETTING_KEYS.APP_ID, value: inputs.AlipayAppID });
      }

      if (
        inputs.AlipayAppPrivateKey !== '' &&
        inputs.AlipayAppPrivateKey !== SENSITIVE_PLACEHOLDER
      ) {
        options.push({
          key: SETTING_KEYS.APP_PRIVATE_KEY,
          value: inputs.AlipayAppPrivateKey,
        });
      }

      if (
        inputs.AlipayPublicKey !== '' &&
        inputs.AlipayPublicKey !== SENSITIVE_PLACEHOLDER
      ) {
        options.push({
          key: SETTING_KEYS.PUBLIC_KEY,
          value: inputs.AlipayPublicKey,
        });
      }

      if (inputs.AlipayAESKey !== '' && inputs.AlipayAESKey !== SENSITIVE_PLACEHOLDER) {
        options.push({ key: SETTING_KEYS.AES_KEY, value: inputs.AlipayAESKey });
      }

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);

      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        setOriginInputs({ ...inputs });
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text='支付宝官方支付设置'>
          <Text>通过支付宝开放平台企业应用对接电脑网站支付，支持充值与订阅。</Text>
          <Banner
            type='info'
            description={`异步通知地址：${
              props.options.ServerAddress
                ? props.options.ServerAddress.replace(/\/$/, '')
                : '网站地址'
            }/api/user/alipay/notify`}
          />
          <Banner
            type='warning'
            description='正式环境需企业支付宝账号并签约「电脑网站支付」；建议先在沙箱环境联调。'
          />
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipayEnabled'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label='启用支付宝支付'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipayIsSandbox'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label='使用沙箱环境'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayAppID'
                label='支付宝 AppID'
                placeholder='例如：2024XXXXXXXXXX'
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayAppPrivateKey'
                label='应用私钥（PKCS8 / RSA2）'
                placeholder='敏感信息不会发送到前端显示；留空则保留原值，输入新值会覆盖。'
                autosize
                rows={6}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayPublicKey'
                label='支付宝公钥'
                placeholder='敏感信息不会发送到前端显示；留空则保留原值，输入新值会覆盖。'
                autosize
                rows={6}
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayAESKey'
                label='AES 密钥（内容加密）'
                placeholder='敏感信息不会发送到前端显示；留空则保留原值，输入新值会覆盖。'
                autosize
                rows={3}
              />
            </Col>
          </Row>
          <Button onClick={submitAlipaySetting} style={{ marginTop: 16 }}>
            更新支付宝设置
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
