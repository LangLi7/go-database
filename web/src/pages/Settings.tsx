import { useState } from 'react'
import { Card, Form, Input, Button, Typography, message, Space } from 'antd'
import { LockOutlined } from '@ant-design/icons'
import { api } from '../api/client'

const { Title } = Typography

export default function Settings() {
  const [pwForm] = Form.useForm()
  const [pwLoading, setPwLoading] = useState(false)

  const handlePasswordChange = async (vals: { old_password: string; new_password: string }) => {
    setPwLoading(true)
    const r = await api.changePassword(vals.old_password, vals.new_password)
    if (r.success) { message.success('Password changed'); pwForm.resetFields() }
    else message.error(r.error?.message || 'Failed')
    setPwLoading(false)
  }

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>Settings</Title>
      <Card title={<><LockOutlined /> Change Password</>} style={{ maxWidth: 500 }}>
        <Form form={pwForm} layout="vertical" onFinish={handlePasswordChange}>
          <Form.Item name="old_password" label="Current Password" rules={[{ required: true }]}><Input.Password /></Form.Item>
          <Form.Item name="new_password" label="New Password" rules={[{ required: true, min: 6 }]}><Input.Password /></Form.Item>
          <Button type="primary" htmlType="submit" loading={pwLoading}>Change Password</Button>
        </Form>
      </Card>
    </div>
  )
}
