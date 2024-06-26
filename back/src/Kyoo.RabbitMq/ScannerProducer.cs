// Kyoo - A portable and vast media library solution.
// Copyright (c) Kyoo.
//
// See AUTHORS.md and LICENSE file in the project root for full license information.
//
// Kyoo is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// Kyoo is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Kyoo. If not, see <https://www.gnu.org/licenses/>.

using System.Text;
using System.Text.Json;
using Kyoo.Abstractions.Controllers;
using Kyoo.Abstractions.Models;
using Kyoo.Utils;
using RabbitMQ.Client;

namespace Kyoo.RabbitMq;

public class ScannerProducer : IScanner
{
	private readonly IModel _channel;

	public ScannerProducer(IConnection rabbitConnection)
	{
		_channel = rabbitConnection.CreateModel();
		_channel.QueueDeclare("scanner", exclusive: false, autoDelete: false);
	}

	private IRepository<T>.ResourceEventHandler _Publish<T>(
		string exchange,
		string type,
		string action
	)
		where T : IResource, IQuery
	{
		return (T resource) =>
		{
			Message<T> message =
				new()
				{
					Action = action,
					Type = type,
					Value = resource,
				};
			_channel.BasicPublish(
				exchange,
				routingKey: message.AsRoutingKey(),
				body: message.AsBytes()
			);
			return Task.CompletedTask;
		};
	}

	public Task SendRefreshRequest(string kind, Guid id)
	{
		var message = new
		{
			Action = "refresh",
			Kind = kind.ToLowerInvariant(),
			Id = id
		};
		var body = Encoding.UTF8.GetBytes(JsonSerializer.Serialize(message, Utility.JsonOptions));
		_channel.BasicPublish("", routingKey: "scanner", body: body);
		return Task.CompletedTask;
	}
}
