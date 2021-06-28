using System.Threading.Tasks;
using Kyoo.Controllers;
using Kyoo.Models;
using Xunit;
using Xunit.Abstractions;

namespace Kyoo.Tests.Library
{
	namespace SqLite
	{
		public class LibraryItemTest : ALibraryItemTest
		{
			public LibraryItemTest(ITestOutputHelper output)
				: base(new RepositoryActivator(output)) { }
		}
	}


	namespace PostgreSQL
	{
		[Collection(nameof(Postgresql))]
		public class LibraryItemTest : ALibraryItemTest
		{
			public LibraryItemTest(PostgresFixture postgres, ITestOutputHelper output)
				: base(new RepositoryActivator(output, postgres)) { }
		}
	}
	
	public abstract class ALibraryItemTest
	{
		private readonly ILibraryItemRepository _repository;
		
		public ALibraryItemTest(RepositoryActivator repositories)
		{
			_repository = repositories.LibraryManager.LibraryItemRepository;
		}

		[Fact]
		public async Task CountTest()
		{
			Assert.Equal(2, await _repository.GetCount());
		}
		
		[Fact]
		public async Task GetShowTests()
		{
			LibraryItem expected = new(TestSample.Get<Show>());
			LibraryItem actual = await _repository.Get(1);
			KAssert.DeepEqual(expected, actual);
		}
		
		[Fact]
		public async Task GetCollectionTests()
		{
			LibraryItem expected = new(TestSample.Get<Show>());
			LibraryItem actual = await _repository.Get(-1);
			KAssert.DeepEqual(expected, actual);
		}
	}
}